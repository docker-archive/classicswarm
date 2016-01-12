package task

import "sync"

type launcher interface {
	LaunchTask(t *Task) bool
}

// Tasks is a simple map of tasks
type Tasks struct {
	sync.Mutex

	cluster launcher
	Tasks   map[string]*Task
}

// NewTasks returns a new tasks
func NewTasks(cluster launcher) *Tasks {
	return &Tasks{
		Tasks:   make(map[string]*Task),
		cluster: cluster,
	}

}

// Add tries to Do the Task, if it's not possible, add the Task to the tasks for future tries
func (t *Tasks) Add(task *Task) {
	if !t.cluster.LaunchTask(task) {
		t.Lock()
		t.Tasks[task.ID()] = task
		t.Unlock()
	}
}

// Remove an Task from the tasks
func (t *Tasks) Remove(tasks ...*Task) {
	t.Lock()
	t.remove(tasks...)
	t.Unlock()
}

// Process tries to Do all the Tasks in the tasks and remove the Tasks successfully done
func (t *Tasks) Process() {
	t.Lock()
	toRemove := []*Task{}
	for _, task := range t.Tasks {
		if t.cluster.LaunchTask(task) {
			toRemove = append(toRemove, task)
		}
	}

	t.remove(toRemove...)
	t.Unlock()
}

func (t *Tasks) remove(tasks ...*Task) {
	for _, task := range tasks {
		task.Stop()
		delete(t.Tasks, task.ID())
	}
}
