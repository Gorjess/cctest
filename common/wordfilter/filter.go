package wordfilter

import (
	"cloudcadetest/common/task"
	"cloudcadetest/framework/module"
	"strconv"
)

type Filter struct {
	srvMod   *module.ServerMod
	tasks    *task.Pool
	id       int64
	trieNode *Trie
}

func New(sm *module.ServerMod, id int64) *Filter {
	f := &Filter{
		id:       id,
		trieNode: NewTrie(),
		srvMod:   sm,
		tasks:    task.NewTaskPool(sm, 0, 0),
	}
	f.trieNode.InsertFile("list.txt")
	return f
}

func (f *Filter) check(content string) string {
	return content
	//return f.trieNode.Replace(content)
}

func (f *Filter) Check(content string, onFinish func(newStr string)) {
	newStr := content
	f.tasks.AddTask(
		func() {
			newStr = f.check(content)
		},
		func() {
			if onFinish != nil {
				onFinish(newStr)
			}
		},
		strconv.FormatInt(f.id, 10),
	)
}
