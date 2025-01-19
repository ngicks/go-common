package pkg2

import (
	"github.com/ngicks/go-common/serr/internal/callstack/pkg3"
)

func F_4_0() error {
	return F_4_1()
}

func F_4_1() error {
	return F_4_2()
}

func F_4_2() error {
	return F_4_3()
}

func F_4_3() error {
	return F_4_4()
}

func F_4_4() error {
	return F_4_5()
}

func F_4_5() error {
	return F_4_6()
}

func F_4_6() error {
	return F_4_7()
}

func F_4_7() error {
	return F_4_8()
}

func F_4_8() error {
	return F_4_9()
}

func F_4_9() error {
	return pkg3.F_0_0()
}
