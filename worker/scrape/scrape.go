package scrape

type Task struct {
	ID       string   `json:"id"`
	Url      string   `json:"url"`
	Response string   `json:"response"`
	Error    []string `json:"error"`
	Browser  bool     `json:"browser"`
}

func Handler(task *Task) error {
	if task.Browser {
		return HandlerBrowser(task)
	} else {
		return HandlerHttpClient(task)
	}
}

func Init() {
	InitHttpClient()
	InitBrowser()
}

func Close() {
	CloseBrowser()
}
