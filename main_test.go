package main

import (
	"net/smtp"
	"testing"
)

func TestAuthLogin(t *testing.T) {

	username := "testuser"
	password := "testpass"

	a := loginAuth(username, password)

	server := smtp.ServerInfo{}

	s, b, err := a.Start(&server)
	if err != nil {
		t.Error("err != null : %v", err.Error())
	}

	stringWant := "LOGIN"
	if s != stringWant || b != nil || err != nil {
		t.Error("a.Start() == (%v,%v,%v), want (%v,%v,%v)", s, nil, stringWant, b, err)
	}

	for _, c := range []struct {
		in   string
		want string
		more bool
		err  error
	}{
		{"username:", username, true, nil},
		{"Username:", username, true, nil},
		{" username:", username, true, nil},
		{"username: ", username, true, nil},
		{" username: ", username, true, nil},

		{"password:", password, true, nil},
		{"Password:", password, true, nil},
		{" password:", password, true, nil},
		{"password: ", password, true, nil},
		{" password: ", password, true, nil},

		{"", "", true, unexpectedSE},
		{" kjsdf ", "", true, unexpectedSE},

		{"", "", false, nil},
	} {
		got, err := a.Next([]byte(c.in), c.more)
		if string(got) != c.want || err != c.err {
			t.Errorf("a.Next(%v) == (%v,%v), want (%v,%v)", c.in, string(got), err, c.want, c.err)
		}

	}

}
