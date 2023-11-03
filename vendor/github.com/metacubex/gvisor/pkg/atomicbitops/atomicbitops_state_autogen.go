// automatically generated by stateify.

//go:build (amd64 || arm64) && !amd64 && !arm64
// +build amd64 arm64
// +build !amd64
// +build !arm64

package atomicbitops

import (
	"github.com/metacubex/gvisor/pkg/state"
)

func (f *Float64) StateTypeName() string {
	return "pkg/atomicbitops.Float64"
}

func (f *Float64) StateFields() []string {
	return []string{
		"bits",
	}
}

func (f *Float64) beforeSave() {}

// +checklocksignore
func (f *Float64) StateSave(stateSinkObject state.Sink) {
	f.beforeSave()
	stateSinkObject.Save(0, &f.bits)
}

func (f *Float64) afterLoad() {}

// +checklocksignore
func (f *Float64) StateLoad(stateSourceObject state.Source) {
	stateSourceObject.Load(0, &f.bits)
}

func (b *Bool) StateTypeName() string {
	return "pkg/atomicbitops.Bool"
}

func (b *Bool) StateFields() []string {
	return []string{
		"Uint32",
	}
}

func (b *Bool) beforeSave() {}

// +checklocksignore
func (b *Bool) StateSave(stateSinkObject state.Sink) {
	b.beforeSave()
	stateSinkObject.Save(0, &b.Uint32)
}

func (b *Bool) afterLoad() {}

// +checklocksignore
func (b *Bool) StateLoad(stateSourceObject state.Source) {
	stateSourceObject.Load(0, &b.Uint32)
}

func init() {
	state.Register((*Float64)(nil))
	state.Register((*Bool)(nil))
}
