package filter

import (
	"cloudcadetest/common/containers/trie"
	"cloudcadetest/common/task"
	"cloudcadetest/framework/module"
	"strconv"
)

type IFilterSkeleton interface {
	GetServerModule() *module.ServerMod
	GetID() int64
	GetWordListFilePath() string
}

type Filter struct {
	srvMod   *module.ServerMod
	tasks    *task.Pool
	id       int64
	trieNode *trie.Trie
}

func New(ifs IFilterSkeleton) *Filter {
	f := &Filter{
		id:       ifs.GetID(),
		trieNode: trie.New(),
	}
	if ifs.GetServerModule() != nil {
		f.tasks = task.NewTaskPool(ifs.GetServerModule(), 0, 0)
	}
	f.trieNode.InsertFile(ifs.GetWordListFilePath())
	return f
}

func (f *Filter) check(content string) string {
	//log.Release("content:%s, passed:%t", content, !f.trieNode.HasDirty(content))
	return f.trieNode.Replace(content)
}

func (f *Filter) Check(content string, onFinish func(newStr string)) {
	var (
		safeFinish = func(s string) {
			if onFinish != nil {
				onFinish(s)
			}
		}
	)

	if f.tasks != nil {
		newStr := content
		f.tasks.AddTask(
			func() {
				newStr = f.check(content)
			},
			func() {
				safeFinish(newStr)
			},
			strconv.FormatInt(f.id, 10),
		)
	} else {
		safeFinish(f.check(content))
	}
}
