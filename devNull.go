package main

// Dummy discard, satisfies io.Writer without importing io or os.
// http://play.golang.org/p/5LIA41Iqfp
type DevNull struct{}

func (DevNull) Write(p []byte) (int, error) {
	return len(p), nil
}
