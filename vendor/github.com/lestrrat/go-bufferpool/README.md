# go-bufferpool

Very simple bytes.Buffer pool using sync.Pool.

I got tired of writing the same syn.Pool for byte.Buffer objects.

# SYNOPSIS


```
import "github.com/lestrrat/go-bufferpool"

var pool = bufferpool.New()

func main() {
    buf := pool.Get()
    defer pool.Release(buf)

    // ...
}
```
