package goja

import (
	"testing"
)

func TestPromise(t *testing.T) {
	r := New()
	r.Set(`err`, func(args ...interface{}) {
		t.Fatal(args...)
	})
	_, err := r.RunString(`
function check(ok,msg){
	if(!ok){
		if(msg){
			err("not pass -> "+msg)
		}else{
			err("not pass")
		}
	}
}
var p = new Promise((resolve,reject)=>{
	resolve(1)
}).then((v)=>{
	check(v==1,'then not equal')
},(e)=>{
	check(false,'unexpected catch')
})
check(p instanceof Promise,'instanceof false')

Promise.reject(123).then(() => {
    check(false,'unexpected then')
}).then((v) => {
	check(false,'err then')
}, (e) => {
	 check(e==123,'catch not equal')
})
Promise.reject(123).then(() => {
    check(false,'unexpected then')
},(e)=>{
	check(e==123,'catch not equal')
	return 456
}).then((v) => {
	check(v==456,'then not equal')
}, (e) => {
	check(false,'unexpected catch')
})

new Promise((resolve, reject) => {
    throw 123
}).catch((e) => {
    check(typeof e==="number")
})
Promise.resolve(123).then((v)=>{
	throw v.toString()
}).catch((e) => {
    check(typeof e==="string")
})
`)
	if err != nil {
		t.Fatal(err)
	}
}
