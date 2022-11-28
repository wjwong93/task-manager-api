package service

import (
	"net/http"
	"crypto/sha256"
	"strconv"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	database "todolist.go/db"
)

func NewUserForm(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "new_user_form.html", gin.H{"Title": "Register user"})
}

func hash(pw string) []byte {
    const salt = "todolist.go#"
    h := sha256.New()
    h.Write([]byte(salt))
    h.Write([]byte(pw))
    return h.Sum(nil)
}

func RegisterNewUser(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	password2 := ctx.PostForm("password2")
	
	switch {
	case username == "": 
		Error(http.StatusBadRequest, "Username is not provided")(ctx)
		return
	case password == "":
		Error(http.StatusBadRequest, "Password is not provided")(ctx)
		return
	case password2 == "":
		Error(http.StatusBadRequest, "Please re-enter password")(ctx)
		return
	case len(password) <= 8:
		Error(http.StatusBadRequest, "Password must be more than 8 characters long")(ctx)
		return
	}

	_, err := strconv.Atoi(password)
	if err == nil {
		Error(http.StatusBadRequest, "Password cannot only contain numerals")(ctx)
		return
	}

	if password != password2 {
		Error(http.StatusBadRequest, "Passwords do not match")(ctx)
		return
	}

	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var duplicate int
	err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name = ?", username)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	if duplicate > 0 {
		Error(http.StatusBadRequest, "Username is already taken")(ctx)
		return
	}

	result, err := db.Exec("INSERT INTO users(name, password) VALUES (?, ?)", username, hash(password))
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	id, _ := result.LastInsertId()
	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE id = ?", id)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	session := sessions.Default(ctx)
	session.Set(userkey, user.ID)
	session.Save()

	// ctx.SetCookie("user-name", user.Name, 3600, "/", "localhost:8081", false, false)
	ctx.Status(http.StatusCreated)
}

func LoginForm(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "login.html", gin.H{ "Title": "Login" })
}

const userkey = "user"

func Login(ctx *gin.Context) {
	username, exist := ctx.GetPostForm("username")
	if !exist {
		Error(http.StatusBadRequest, "Username not provided")(ctx)
		return
	}
	password, exist := ctx.GetPostForm("password")
	if !exist {
		Error(http.StatusBadRequest, "Password not provided")(ctx)
		return
	}

	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE is_deleted = 0 AND name = ?", username)
	if err != nil {
		// ctx.HTML(http.StatusBadRequest, "login.html", gin.H{ "Title": Login, "Username": username, "Error": "No such user" })
		Error(http.StatusBadRequest, "User not found")(ctx)
		return
	}

	if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
		Error(http.StatusBadRequest, "Incorrect password")(ctx)
		return
	}

	session := sessions.Default(ctx)
	session.Set(userkey, user.ID)
	session.Save()

	// ctx.Redirect(http.StatusFound, "/list")
	ctx.Status(http.StatusOK)
}

func Logout(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Clear()
	session.Options(sessions.Options{MaxAge: -1})
	session.Save()
	// ctx.Status(http.StatusFound)
}

func GetUsername(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")

	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE is_deleted = 0 AND id = ?", userID)
	if err != nil {
		Error(http.StatusBadRequest, "User not found")(ctx)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"name": user.Name})
}

func ChangeUsername(ctx *gin.Context) {
	newUsername, exist := ctx.GetPostForm("new_username")
	if !exist {
		Error(http.StatusBadRequest, "New username not provided")(ctx)
		return
	}

	password, exist := ctx.GetPostForm("password")
	if !exist {
		Error(http.StatusBadRequest, "Password not provided")(ctx)
		return
	}

	userID := sessions.Default(ctx).Get("user")

	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var duplicate int
	err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name = ?", newUsername)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	if duplicate > 0 {
		Error(http.StatusBadRequest, "Username is already taken")(ctx)
		return
	}

	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE is_deleted = 0 AND id = ?", userID)
	if err != nil {
		Error(http.StatusBadRequest, "User not found")(ctx)
		return
	}

	if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
		Error(http.StatusBadRequest, "Incorrect password")(ctx)
		return
	}

	_, err = db.Exec("UPDATE users SET name = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	ctx.Status(http.StatusOK)
}

func ChangePassword(ctx *gin.Context) {
	curr_pw, exist := ctx.GetPostForm("curr_password")
	if !exist {
		Error(http.StatusBadRequest, "Current Password not provided")(ctx)
		return
	}
	new_pw, exist := ctx.GetPostForm("new_password")
	if !exist {
		Error(http.StatusBadRequest, "New Password not provided")(ctx)
		return
	}
	new_pw2, exist := ctx.GetPostForm("new_password2")
	if !exist {
		Error(http.StatusBadRequest, "New Password not re-entered")(ctx)
		return
	}

	if len(new_pw) <= 8 {
		Error(http.StatusBadRequest, "Password must be more than 8 characters long")(ctx)
		return
	}

	_, err := strconv.Atoi(new_pw)
	if err == nil {
		Error(http.StatusBadRequest, "Password cannot only contain numerals")(ctx)
		return
	}

	if curr_pw == new_pw {
		Error(http.StatusBadRequest, "Password is not changed")(ctx)
		return
	}

	if new_pw != new_pw2 {
		Error(http.StatusBadRequest, "Passwords do not match")(ctx)
		return
	}

	userID := sessions.Default(ctx).Get("user")

	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE is_deleted = 0 AND id = ?", userID)
	if err != nil {
		Error(http.StatusBadRequest, "User not found")(ctx)
		return
	}

	if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(curr_pw)) {
		Error(http.StatusBadRequest, "Incorrect password")(ctx)
		return
	}

	_, err = db.Exec("UPDATE users SET password = ? WHERE is_deleted = 0 AND id = ?", hash(new_pw), userID)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	ctx.Status(http.StatusOK)
}

func DeleteUser(ctx *gin.Context) {
	pw, exist := ctx.GetPostForm("password")
	if !exist {
		Error(http.StatusBadRequest, "Password not provided")(ctx)
		return
	}
	userID := sessions.Default(ctx).Get("user")
	
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	var user database.User
	err = db.Get(&user, "SELECT id, name, password FROM users WHERE is_deleted = 0 AND id = ?", userID)
	if err != nil {
		Error(http.StatusBadRequest, "User not found")(ctx)
		return
	}

	if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(pw)) {
		Error(http.StatusBadRequest, "Incorrect password")(ctx)
		return
	}

	_, err = db.Exec("UPDATE users SET is_deleted = 1 WHERE is_deleted = 0 AND id = ?", userID)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	ctx.Status(http.StatusOK)
}