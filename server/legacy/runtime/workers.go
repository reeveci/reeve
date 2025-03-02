package runtime

// Mapping of worker names to their corresponding configurations
type WorkerMap map[string]ServerWorkerConfig

type ServerWorkerConfig struct {
	WorkerConfig

	// Hash of the worker secret
	Secret string `json:"secret"`
}

type WorkerConfig struct {
	// Specifies which worker groups this worker is assigned to and how many pipelines may be executed concurrently
	WorkerGroups map[string]int `json:"workerGroups"`

	// Status update configuration
	Status struct {
		// Defines the interval in seconds at which the worker should report its online status to the server
		UpdateInterval uint `json:"updateInterval"`

		// Defines how many seconds the update interval may be exceeded for a worker to be still considered as operating normally
		DelayCompensation uint `json:"delayCompensation"`

		// Defines how many seconds both the update interval and the delay compensation may be exceeded before a worker is considered to be down
		GracePeriod uint `json:"gracePeriod"`
	} `json:"status"`
}
