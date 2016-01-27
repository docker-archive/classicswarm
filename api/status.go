package api

// StatusHandler allows the API to display extra information on docker info.
type StatusHandler interface {
	// Info provides key/values to be added to docker info.
	Status() [][2]string
}
