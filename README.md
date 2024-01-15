# related-tasks-handler-go
A Golang repository that handles hierarchically related tasks

Just present for all your tasks:
* their id
* the task id's they have to wait for
* a task handler

### Usage:
```go
import (
    "errors"
    "fmt"
    "time"
	rth "github.com/caselongo/related-tasks-handler-go"
)

func main(){
	// YOUR TASK AND THEIR RELATIONS
    tasks := []rth.Task{
        {Id:      "a", WaitFor: nil},
        {Id:      "b", WaitFor: []string{"a"},},
        {Id:      "c", WaitFor: []string{"a"},},
        {Id:      "d", WaitFor: []string{"a", "b", "c"},},
    }

	// YOUR TASK HANDLER FUNC
    handlerFunc := func (id string) error {
        fmt.Println("executing:", id)
        
        time.Sleep(2 * time.Second)
        
        return nil
    }
    
    h, err := rth.NewHandler(handlerFunc, tasks...)
    if err != nil {
        log.Fatal(err)
    }
    
    err = h.Run()
    if err != nil {
        log.Fatal(err)
    }
}
```