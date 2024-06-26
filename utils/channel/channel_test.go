package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type data struct {
	val int
}

func TestChannel(t *testing.T) {
	t.Parallel()

	testContext, testCancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(testCancel)

	d := &data{5}

	t.Run("reader", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(testContext)
		t.Cleanup(cancel)

		chan1 := make(chan *data, 10)
		c := NewReader(ctx, chan1)
		require.Equal(t, ctx, c.Context())

		chan1 <- d
		val, ok := c.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, ok = c.Read()
			require.False(t, ok)
			require.Nil(t, val)
		}()

		// Make sure we are waiting on the channel.
		time.Sleep(time.Second)
		cancel()
		wg.Wait()
	})

	t.Run("writer", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(testContext)
		t.Cleanup(cancel)

		chan1 := make(chan *data, 1)
		c := NewWriter(ctx, chan1)
		require.Equal(t, ctx, c.Context())
		require.True(t, c.Write(d))

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.False(t, c.Write(d))
		}()

		// Make sure we are waiting on the channel.
		time.Sleep(time.Second)
		cancel()
		wg.Wait()

		require.Len(t, chan1, 1)
		require.Equal(t, d, <-chan1)
	})

	t.Run("reader-writer", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(testContext)
		t.Cleanup(cancel)

		chan1 := make(chan *data)
		c := NewReaderWriter(ctx, chan1)
		require.Equal(t, ctx, c.Context())

		go func() {
			require.True(t, c.Write(d))
		}()

		val, ok := c.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.False(t, c.Write(d))
		}()

		// Make sure we are waiting on the channel.
		time.Sleep(time.Second)
		cancel()
		wg.Wait()

		val, ok = c.Read()
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("with-context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(testContext)
		t.Cleanup(cancel)

		chan1 := make(chan *data, 10)
		c := NewReaderWriter(ctx, chan1)
		require.Equal(t, ctx, c.Context())

		require.True(t, c.Write(d))
		val, ok := c.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		ctx2, cancel2 := context.WithCancel(ctx)
		t.Cleanup(cancel2)
		c2 := c.WithContext(ctx2)
		require.Equal(t, ctx, c.Context())
		require.Equal(t, ctx2, c2.Context())

		require.True(t, c.Write(d))
		require.True(t, c2.Write(d))

		val, ok = c.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		val, ok = c2.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		cancel2()
		require.True(t, c.Write(d))
		require.False(t, c2.Write(d))

		val, ok = c.Read()
		require.True(t, ok)
		require.Equal(t, d, val)

		val, ok = c2.Read()
		require.False(t, ok)
		require.Nil(t, val)
	})
}
