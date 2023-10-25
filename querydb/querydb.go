package querydb

import (
	"context"
	"errors"
	"fmt"

	"github.com/0sm1les/gopherbb/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbpool *pgxpool.Pool

func Connect(creds string, address string, database string) error {
	URL := fmt.Sprintf("postgres://%s@%s/%s", creds, address, database)
	var err error
	dbpool, err = pgxpool.New(context.Background(), URL)
	if err != nil {
		return err
	}
	return nil
}

func UserExists(username models.Username) int32 {
	var user_id int32
	err := dbpool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = $1", username).Scan(&user_id)
	if err != nil {
		return -1
	}
	return user_id
}

func CreateUser(user models.Username, hash models.Hash) error {
	_, err := dbpool.Exec(context.Background(), "INSERT INTO users (username, password, date_joined) VALUES ($1, $2, NOW())", user, hash)
	return err
}

func Authenticate(user models.Username, hash models.Hash) (int32, error) {
	var user_id int32
	err := dbpool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = $1 AND password = $2", user, hash).Scan(&user_id)
	if err != nil {
		return -1, err
	}
	return user_id, nil
}

func Userinfo(user_id int32) (models.User, error) {
	var userinfo models.User

	err := dbpool.QueryRow(context.Background(), "SELECT id, role, profile_pic, username ,password, bio, user_fg_color, user_bg_color, date_joined FROM users WHERE id = $1", user_id).Scan(
		&userinfo.Id,
		&userinfo.Role,
		&userinfo.Profile_pic,
		&userinfo.Username,
		&userinfo.Password,
		&userinfo.Bio,
		&userinfo.User_fg_color,
		&userinfo.User_bg_color,
		&userinfo.Date_Joined,
	)
	if err != nil {
		return userinfo, err
	}

	return userinfo, nil
}

func SetBio(user_id int32, bio string) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE users SET bio = $1 WHERE id = $2", bio, user_id)
	return err
}

func SetColor(user_id int32, elm string, hex string) error {
	if elm == "fg" {
		_, err := dbpool.Exec(context.Background(), "UPDATE users SET user_fg_color = $1 WHERE id = $2", hex, user_id)
		return err
	} else if elm == "bg" {
		_, err := dbpool.Exec(context.Background(), "UPDATE users SET user_bg_color = $1 WHERE id = $2", hex, user_id)
		return err
	} else {
		return errors.New("invalid attribute")
	}
}

func SetPFP(user_id int32, filename string) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE users SET profile_pic = $1 WHERE id = $2", filename, user_id)
	return err
}

// returns post id and error
func NewPost(user_id int32, section string, status string, title string, md string, html string) (int32, error) {
	var post_id int32
	err := dbpool.QueryRow(context.Background(), "INSERT INTO posts (poster,section, status, title, md, html, time_posted) VALUES ($1,$2,$3,$4,$5,$6,NOW()) RETURNING id",
		user_id,
		section,
		status,
		title,
		md,
		html).Scan(&post_id)
	if err != nil {
		return -1, err
	}
	return post_id, nil
}

func GetPost(post_id int32) (models.Post, error) {
	var post models.Post
	err := dbpool.QueryRow(context.Background(), "SELECT id, poster, status, title, section, md, html, time_posted FROM posts WHERE id = $1",
		post_id).Scan(&post.Pid,
		&post.Uid,
		&post.Status,
		&post.Title,
		&post.Section,
		&post.Md,
		&post.Html,
		&post.Time_posted)
	return post, err
}

func UserPosts(user_id int32, status string) ([]models.PostListing, error) {
	var posts []models.PostListing
	results, err := dbpool.Query(context.Background(), "SELECT id, title, section,time_posted FROM posts WHERE poster = $1 AND status = $2", user_id, status)
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var post models.PostListing
		if err := results.Scan(&post.Pid, &post.Title, &post.Section, &post.Time_posted); err != nil {
			return nil, err
		}
		posts = append(posts, post)

	}
	return posts, nil
}

func RecentUserPosts(user_id int32) ([]models.PostListing, error) {
	var posts []models.PostListing
	results, err := dbpool.Query(context.Background(), "SELECT id, title, section,time_posted FROM posts WHERE poster = $1 AND status = $2 ORDER BY time_posted DESC LIMIT 4", user_id, "posted")
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var post models.PostListing
		if err := results.Scan(&post.Pid, &post.Title, &post.Section, &post.Time_posted); err != nil {
			return nil, err
		}
		posts = append(posts, post)

	}
	return posts, nil
}

func UpdatePost(post_id int32, title string, md string, html string, section string) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE posts SET title = $1, md = $2, html = $3, section = $4 WHERE id = $5",
		title,
		md,
		html,
		section,
		post_id)
	return err
}

func UpdatePostStatus(post_id int32, status string) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE posts SET status = $1 WHERE id = $2", status, post_id)
	return err
}

func GetSectionPosts(section string) ([]models.PostListing, error) {
	var posts []models.PostListing
	results, err := dbpool.Query(context.Background(), "SELECT id, poster, title, time_posted FROM posts WHERE status = $1 AND section = $2", "posted", section)
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var post models.PostListing
		err = results.Scan(&post.Pid, &post.Uid, &post.Title, &post.Time_posted)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func GetUser(user_id int32) (models.Userlisted, error) {
	var user models.Userlisted
	err := dbpool.QueryRow(context.Background(), "SELECT username, role, user_fg_color, user_bg_color FROM users WHERE id = $1", user_id).Scan(&user.Username,
		&user.Role,
		&user.User_fg_color,
		&user.User_bg_color)
	return user, err
}

func PostComment(user_id int32, parent_post int32, comment_post int32, md string, html string) (int32, error) {
	var comment_id int32
	var err error
	if comment_id != -1 {
		err = dbpool.QueryRow(context.Background(), "INSERT into comments (poster, parent_post, parent_comment, md, html, time_posted) VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id",
			user_id,
			parent_post,
			comment_post,
			md,
			html).Scan(&comment_id)
	} else if comment_id == -1 {
		err = dbpool.QueryRow(context.Background(), "INSERT into comments (poster, parent_post, md, html, time_posted) VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id",
			user_id,
			parent_post,
			md,
			html).Scan(&comment_id)
	}
	return comment_id, err
}

func GetComments(post_id int32) ([]models.Comment, error) {
	var comments []models.Comment
	results, err := dbpool.Query(context.Background(), "SELECT id, poster, parent_post, parent_comment, html, time_posted FROM comments WHERE parent_post = $1 AND status = $2", post_id, "posted")
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var comment models.Comment
		err = results.Scan(&comment.Cid, &comment.User_id, &comment.Parent_post, &comment.Comment_post, &comment.Html, &comment.Time_posted)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func LikeUnlike(user_id int32, post_id int32) error {
	var check int32
	err := dbpool.QueryRow(context.Background(), "SELECT id FROM likes WHERE liked_by = $1 AND post = $2", user_id, post_id).Scan(&check)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return err
		}
	}
	if check != 0 {
		_, err = dbpool.Exec(context.Background(), "DELETE FROM likes WHERE id = $1", check)
		return err
	}
	_, err = dbpool.Exec(context.Background(), "INSERT INTO likes (post, liked_by, time_liked) VALUES ($1, $2, NOW())", post_id, user_id)
	return err
}

func Liked(user_id int32, post_id int32) (bool, error) {
	var check int32
	err := dbpool.QueryRow(context.Background(), "SELECT id FROM likes WHERE liked_by = $1 AND post = $2", user_id, post_id).Scan(&check)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return false, err
		}
	}
	if check != 0 {
		return true, nil
	}
	return false, nil
}

func Likes(user_id int32) ([]models.PostListing, error) {
	var posts []models.PostListing
	results, err := dbpool.Query(context.Background(), "SELECT p.id, p.poster ,p.title, p.section, p.time_posted FROM posts p INNER JOIN likes l ON p.id = l.post WHERE l.liked_by = $1 AND p.status = $2", user_id, "posted")
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var post models.PostListing
		err = results.Scan(&post.Pid, &post.Uid, &post.Title, &post.Section, &post.Time_posted)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func GetPostOP(pid int32) (int32, string, string, error) {
	var uid int32
	var section string
	var title string
	err := dbpool.QueryRow(context.Background(), "SELECT poster, section, title FROM posts WHERE id = $1", pid).Scan(&uid, &section, &title)
	return uid, section, title, err
}

func GetCommentPoster(cid int32) (int32, error) {
	var uid int32
	err := dbpool.QueryRow(context.Background(), "SELECT poster FROM comments WHERE id = $1", cid).Scan(&uid)
	return uid, err
}

func NewNotification(to_uid int32, from_uid int32, message string) error {
	_, err := dbpool.Exec(context.Background(), "INSERT INTO notifications (to_uid, from_uid, msg) VALUES ($1, $2, $3)", to_uid, from_uid, message)
	return err
}

func Notifications(user_id int32) ([]models.Notification, error) {
	var notifications []models.Notification
	results, err := dbpool.Query(context.Background(), "SELECT id, from_uid, msg FROM notifications WHERE to_uid = $1 AND read = $2", user_id, false)
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var notification models.Notification
		err = results.Scan(&notification.Nid, &notification.From_Uid, &notification.Message)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, nil
}

func Search(search_qry string) ([]models.PostListing, error) {
	var posts []models.PostListing

	results, err := dbpool.Query(context.Background(), "SELECT  id, poster, title, time_posted FROM posts WHERE ts @@ phraseto_tsquery('english', $1)", search_qry)
	if err != nil {
		return nil, err
	}
	for results.Next() {
		var post models.PostListing
		err = results.Scan(&post.Pid, &post.Uid, &post.Title, &post.Time_posted)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func DeletePost(pid int32) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE posts SET status = $1 WHERE id = $2", "deleted", pid)
	return err
}

func DeleteReply(cid int32) error {
	_, err := dbpool.Exec(context.Background(), "UPDATE comments SET status = $1 WHERE id = $2", "deleted", cid)
	return err
}
