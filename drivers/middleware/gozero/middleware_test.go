package gozero_test

import (
	"github.com/ulule/limiter/v3/drivers/middleware/gozero"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func TestHTTPMiddleware(t *testing.T) {
	is := require.New(t)

	request, err := http.NewRequest("GET", "/", nil)
	is.NoError(err)
	is.NotNil(request)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, thr := w.Write([]byte("hello"))
		if thr != nil {
			panic(thr)
		}
	})

	store := memory.NewStore()
	is.NotZero(store)

	rate, err := limiter.NewRateFromFormatted("10-M")
	is.NoError(err)
	is.NotZero(rate)

	middleware := gozero.NewMiddleware(limiter.New(store, rate)).Handle(handler)
	is.NotZero(middleware)

	success := int64(10)
	clients := int64(100)

	//
	// Sequential
	//

	for i := int64(1); i <= clients; i++ {

		resp := httptest.NewRecorder()
		middleware.ServeHTTP(resp, request)

		if i <= success {
			is.Equal(resp.Code, http.StatusOK)
		} else {
			is.Equal(resp.Code, http.StatusTooManyRequests)
		}
	}

	//
	// Concurrent
	//

	store = memory.NewStore()
	is.NotZero(store)

	middleware = gozero.NewMiddleware(limiter.New(store, rate)).Handle(handler)
	is.NotZero(middleware)

	wg := &sync.WaitGroup{}
	counter := int64(0)

	for i := int64(1); i <= clients; i++ {
		wg.Add(1)
		go func() {

			resp := httptest.NewRecorder()
			middleware.ServeHTTP(resp, request)

			if resp.Code == http.StatusOK {
				atomic.AddInt64(&counter, 1)
			}

			wg.Done()
		}()
	}

	wg.Wait()
	is.Equal(success, atomic.LoadInt64(&counter))

}
