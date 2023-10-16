package main 

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"os/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)
type Data struct{
	Target string `json:"target"`}

func newTarget(c *gin.Context){
	var data Data;
	if err:=c.ShouldBindJSON(&data);
	err!=nil{
	c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
	return	}
	err := addTarget(data.Target)
    if err != nil {
        log.Println("Error adding target:", err)
		c.String(504,string(err.Code))
		return
    }
	log.Printf(data.Target)
	c.String(200, "Target Added To Database sucessfully")
}

func main(){
	router := gin.Default()
	router.Use(cors.New(cors.Config{AllowOrigins: []string{"*"},AllowHeaders:[]string{"content-type"},ExposeHeaders:[]string{"Content-Length"},}))
	router.POST("/new",newTarget)
	router.Run("localhost:8000")
}
var dbPath string

func init() {
    // Initialize the database path
    currentUser, err := user.Current()
    if err != nil {
        log.Fatal("Error getting current user:", err)
    }
    dbPath = filepath.Join(currentUser.HomeDir, ".autoscout", "autoscout.db")
}

func addTarget(target string) error {
    // Open the SQLite database
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return err
    }
    defer db.Close()

    // Create the 'targets' table if it doesn't exist
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS targets ( url TEXT PRIMARY KEY)`)
    if err != nil {
        return err
    }

    // Insert the target into the 'targets' table
    _, err = db.Exec("INSERT INTO targets (url) VALUES (?)", target)
    if err != nil {
        return err
    }

    log.Println("Target added:", target)
    return nil
}	

