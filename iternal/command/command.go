package command

type Command interface {
	Execute(s string) string
}

func Exec(c Command, uname string) string {
	return c.Execute(uname)
}
