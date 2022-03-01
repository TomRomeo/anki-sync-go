package main

type session struct {
	Username string
	Skey     string
	created  string
}

type dbCard struct {
	Username string
	Id       int
	Nid      int
	Did      int
	Ord      int
	Mod      int
	Usn      int
	Type     int
	Queue    int
	Due      int
	Ivl      int
	Factor   int
	Reps     int
	Lapses   int
	Left     int
	Odue     int
	Odid     int
	Flags    int
	Data     string
}
