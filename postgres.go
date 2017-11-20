package main

import "database/sql"
import "log"
import "fmt"
import _ "github.com/lib/pq"

type PostgresClient struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Db       *sql.DB
}

func (p *PostgresClient) GetDB() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.Dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return db
}

type Class struct {
	ClassId   string
	ClassName string
}

// Returns a list of queryable classes
func (p *PostgresClient) GetAvailableClasses() []*Class {
	sqlStatement := `
	SELECT DISTINCT class_id, class_name FROM classifications`
	rows, err := p.Db.Query(sqlStatement)
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	ret := make([]*Class, 0)
	var (
		classId   string
		className string
	)
	for rows.Next() {
		if err := rows.Scan(&classId, &className); err != nil {
			log.Fatal(err)
		}
		class := &Class{ClassId: classId, ClassName: className}
		ret = append(ret, class)

	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return ret
}

func (p *PostgresClient) GetImageCount() int {
	sqlStatement := `
	SELECT count(*)
	FROM images`
	rows, err := p.Db.Query(sqlStatement)
	defer rows.Close()
	if err != nil {
		panic(err)
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return count
}

// Return all images that belong to this class
func (p *PostgresClient) GetClassMembers(classId string) []string {
	// TODO: classifications should probably be correctly indexed with a foreign key into images
	sqlStatement := `
	SELECT images.filename 
	FROM classifications 
	INNER JOIN images on classifications.image_id=images.image_id
	WHERE class_id=$1`
	rows, err := p.Db.Query(sqlStatement, classId)
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	ret := make([]string, 0)
	var fname string
	for rows.Next() {
		if err := rows.Scan(&fname); err != nil {
			log.Fatal(err)
		}
		ret = append(ret, fname)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return ret
}

func NewPostgresClient(pgHost string, pgPort int, pgUser string,
	pgPassword string, pgDbname string) *PostgresClient {
	p := new(PostgresClient)
	p.Host = pgHost
	p.Port = pgPort
	p.User = pgUser
	p.Password = pgPassword
	p.Dbname = pgDbname
	p.Db = p.GetDB()
	p.Db.SetMaxOpenConns(50)
	return p
}
