package armo_builtins
default hello = false

hello {
    m := input.message
    m == "world"
    k := input.kind
	k == "Pod"
}