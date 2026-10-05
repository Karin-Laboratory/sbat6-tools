package api

type Request struct {
	ID        string `json:"id,omitempty"`
	Operation string `json:"operation"`
}

type Response struct {
	ID       string `json:"id,omitempty"`
	OK       bool   `json:"ok"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}
