package debugger

import "github.com/dop251/goja"

// RefManager maps DAP integer variable references to goja objects or scope data.
// It is only accessed from the server goroutine, so no mutex is needed.
type RefManager struct {
	nextRef int
	refs    map[int]interface{}
}

type scopeEntry struct {
	frameIndex int
	scopeIndex int
	scope      goja.DebugScope
}

type objectEntry struct {
	object *goja.Object
}

// NewRefManager creates a new RefManager.
func NewRefManager() *RefManager {
	return &RefManager{
		refs: make(map[int]interface{}),
	}
}

// AddScope stores a scope and returns a reference ID.
func (rm *RefManager) AddScope(frameIndex, scopeIndex int, scope goja.DebugScope) int {
	rm.nextRef++
	rm.refs[rm.nextRef] = scopeEntry{
		frameIndex: frameIndex,
		scopeIndex: scopeIndex,
		scope:      scope,
	}
	return rm.nextRef
}

// AddObject stores a goja Object and returns a reference ID.
func (rm *RefManager) AddObject(obj *goja.Object) int {
	rm.nextRef++
	rm.refs[rm.nextRef] = objectEntry{object: obj}
	return rm.nextRef
}

// Get retrieves the stored value for a reference ID.
func (rm *RefManager) Get(ref int) (interface{}, bool) {
	v, ok := rm.refs[ref]
	return v, ok
}

// Clear drops all references. Called on each new stop to invalidate
// previous scope/variable references and allow GC of goja objects.
func (rm *RefManager) Clear() {
	rm.refs = make(map[int]interface{})
	rm.nextRef = 0
}
