package models

import (
	"html/template"
	"time"
)

type Username string
type Password string
type Hash string

type User struct {
	Id             int32
	Role           string
	Profile_pic    string
	Username       Username
	Password       Hash
	Bio            string
	User_fg_color  string
	User_bg_color  string
	Date_Joined    time.Time
	Date_formatted string
}

type Userlisted struct {
	Username      Username
	Role          string
	User_fg_color string
	User_bg_color string
}

type Post struct {
	Pid            int32         `json:"pid"`
	Uid            int32         `json:"uid"`
	Status         string        `json:"status"`
	Title          string        `json:"title"`
	Section        string        `json:"section"`
	Md             string        `json:"md"`
	Html           template.HTML `json:"html"`
	Time_posted    time.Time     `json:"time_posted"`
	Time_formatted string        `json:"time_formatted"`
}

type PostListing struct {
	Pid            int32      `json:"pid"`
	Uid            int32      `json:"uid"`
	User           Userlisted `json:"user"`
	Title          string     `json:"title"`
	Status         string     `json:"status"`
	Section        string     `json:"section"`
	Time_posted    time.Time  `json:"time_posted"`
	Time_formatted string     `json:"time_formatted"`
}

type Comment struct {
	Cid          int32         `json:"Cid"`
	Parent_post  int32         `json:"Parent"`
	Comment_post int32         `json:"Comment"`
	User_id      int32         `json:"uid"`
	User         Userlisted    `json:"user"`
	Md           string        `json:"md"`
	Html         template.HTML `json:"html"`
	Time_posted  time.Time     `json:"time_posted"`
}

type Notification struct {
	Nid              int32
	To_Uid           int32
	From_Uid         int32
	From_Uid_Listing Userlisted
	Message          template.HTML
}

type Section struct {
	Section string
	Id      string
}

type Category struct {
	Category string
	Sections []Section
}
