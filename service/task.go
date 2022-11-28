package service

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"github.com/jmoiron/sqlx"
	database "todolist.go/db"
)

// TaskList renders list of tasks in DB
func TaskList(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	fmt.Printf("%d\n", userID)
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// Get query parameter
	kw := ctx.Query("kw")
	status := ctx.Query("completed")
	due_date := ctx.Query("due")
	priority := ctx.QueryArray("priority[]")

	// var is_done bool
	if status != "" {
		_, err = strconv.ParseBool(status)
		if err != nil {
			Error(http.StatusBadRequest, err.Error())(ctx)
			return
		}
	}
	
	query := "SELECT id, title, priority, due_date, created_at, is_done, description FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"

	params := []interface{}{}
	params = append(params, fmt.Sprintf("%d", userID))
	if kw != "" {
		query += " AND title LIKE ?"
		params = append(params, "%"+kw+"%")
	}

	if status != "" {
		query += " AND is_done = ?"
		params = append(params, status)
	}

	if due_date != "" {
		query += " AND due_date = ?"
		params = append(params, due_date)
	}

	if len(priority) > 0 {
		q, p, err := sqlx.In(" AND `priority` IN (?)", priority)
		if err != nil {
			Error(http.StatusInternalServerError, err.Error())(ctx)
			return
		}
		query += q
		params = append(params, p...)
	}

	// Get tasks in DB
	tasks := []database.Task{}
	err = db.Select(&tasks, query, params...) // Use DB#Select for multiple entries
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// Render tasks
	// ctx.HTML(http.StatusOK, "task_list.html", gin.H{"Title": "Task list", "Tasks": tasks, "Kw": kw, "Status": status })
	ctx.JSON(http.StatusOK, tasks)
}

// ShowTask renders a task with given ID
func ShowTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")

	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// parse ID given as a parameter
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// check if valid owner
	var ownershipStatus database.TaskOwner
	err = db.Get(&ownershipStatus, "SELECT * FROM ownership WHERE user_id=? AND task_id=?", userID, id)
	if err != nil {
		Error(http.StatusForbidden, "Forbidden Action")(ctx)
		return
	}

	// Get a task with given ID
	var task database.Task
	err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id) // Use DB#Get for one entry
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Render task
	// ctx.String(http.StatusOK, task.Title)  // Modify it!!
	ctx.HTML(http.StatusOK, "task.html", task)
}

func NewTaskForm(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "form_new_task.html", gin.H{ "Title": "Task registration" })
}

func RegisterTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	title, exist := ctx.GetPostForm("title")
	if !exist {
		Error(http.StatusBadRequest, "No title given")(ctx)
		return
	}
	due_date, exist := ctx.GetPostForm("due_date")
	if !exist {
		Error(http.StatusBadRequest, "No due date given")(ctx)
		return
	}
	priority, _ := ctx.GetPostForm("priority")
	description, _ := ctx.GetPostForm("description")
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	tx := db.MustBegin()
	result, err := tx.Exec("INSERT INTO tasks (title, priority, due_date, description) VALUES (?, ?, ?, ?)", title, priority, due_date, description)
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	_, err = tx.Exec("INSERT INTO ownership (user_id, task_id) VALUES (?, ?)", userID, taskID)
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	tx.Commit()
	// path := "/list"
	// if id, err := result.LastInsertId(); err == nil {
	// 	path = fmt.Sprintf("/task/%d", id)
	// }
	// ctx.Redirect(http.StatusFound, path)
	// ctx.Redirect(http.StatusFound, fmt.Sprintf("/task/%d", taskID))
	ctx.Status(http.StatusCreated)
}

func EditTaskForm(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	var ownershipStatus database.TaskOwner
	err = db.Get(&ownershipStatus, "SELECT * FROM ownership WHERE user_id=? AND task_id=?", userID, id)
	if err != nil {
		Error(http.StatusForbidden, "Forbidden Action")(ctx)
		return
	}
	var task database.Task
	err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	ctx.HTML(
		http.StatusOK, 
		"form_edit_task.html", 
		gin.H{ 
			"Title": fmt.Sprintf("Edit Task %d", task.ID), 
			"Task": task,
		},
	)
}

func UpdateTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
	
	title, exist := ctx.GetPostForm("title")
	if !exist {
		Error(http.StatusBadRequest, "No Title")(ctx)
		return
	}
	priority, exist := ctx.GetPostForm("priority")
	if !exist {
		Error(http.StatusBadRequest, "No Priority")(ctx)
		return
	}
	due_date, exist := ctx.GetPostForm("due_date")
	if !exist {
		Error(http.StatusBadRequest, "No Due Date")(ctx)
		return
	}
	is_done_str, exist := ctx.GetPostForm("is_done")
	if !exist {
		Error(http.StatusBadRequest, "No Status")(ctx)
		return
	}
	is_done, err := strconv.ParseBool(is_done_str)
	if err != nil {
		Error(http.StatusBadRequest, "IsDone is non-boolean")(ctx)
		return
	}
	description, exist := ctx.GetPostForm("description")
	if !exist {
		Error(http.StatusBadRequest, "No Description")(ctx)
		return
	}
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	var ownershipStatus database.TaskOwner
	err = db.Get(&ownershipStatus, "SELECT * FROM ownership WHERE user_id=? AND task_id=?", userID, id)
	if err != nil {
		Error(http.StatusForbidden, "Forbidden Action")(ctx)
		return
	}
	_, err = db.Exec("UPDATE tasks SET title=?, priority=?, due_date=?, is_done=?, description=? WHERE id=?", title, priority, due_date, is_done, description, id)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	// path := fmt.Sprintf("/task/%d", id)
	// ctx.Redirect(http.StatusFound, path)
	ctx.Status(http.StatusOK)
}

func DeleteTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	var ownershipStatus database.TaskOwner
	err = db.Get(&ownershipStatus, "SELECT * FROM ownership WHERE user_id=? AND task_id=?", userID, id)
	if err != nil {
		Error(http.StatusForbidden, "Forbidden Action")(ctx)
		return
	}
	_, err = db.Exec("DELETE tasks, ownership FROM tasks INNER JOIN ownership WHERE id=task_id AND id=?", id)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	// ctx.Redirect(http.StatusFound, "/list")
	ctx.Status(http.StatusOK)
}