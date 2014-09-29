package main


//http://www.golangpatterns.info/concurrency/semaphores
type empty struct {}
type semaphore chan empty

// acquire n resources
func (s semaphore) P(n int) {
	e := empty{}
	for i := 0; i < n; i++ {
		s <- e
	}
}
// release n resources
func (s semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}
//Acquire Single Resource
func (s semaphore) Lock() {
	s.P(1)
}
//Release Single Resource
func (s semaphore) Unlock() {
	s.V(1)
}
