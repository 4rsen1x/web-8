package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1"
	dbname   = "postgres"
)

type Handlers struct {
	dbProvider DatabaseProvider
}

type DatabaseProvider struct {
	db *sql.DB
}

// Обработчики HTTP-запросов

// /count: GET и POST
func (h *Handlers) GetCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.dbProvider.SelectCount()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(count)))
}

func (h *Handlers) PostCount(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	countStr := r.FormValue("count")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		http.Error(w, "это не число", http.StatusBadRequest)
		return
	}

	err = h.dbProvider.UpdateCount(count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
func (dp *DatabaseProvider) SelectCount() (int, error) {
	var count int

	err := dp.db.QueryRow("SELECT value FROM counter WHERE id = 1").Scan(&count)
	if err == sql.ErrNoRows {
		// Если данных нет, инициализируем значение
		_, err = dp.db.Exec("INSERT INTO counter (id, value) VALUES (1, 0)")
		if err != nil {
			return 0, err
		}
		count = 0
	} else if err != nil {
		return 0, err
	}

	return count, nil
}

func (dp *DatabaseProvider) UpdateCount(value int) error {
	_, err := dp.db.Exec("UPDATE counter SET value = value + $1 WHERE id = 1", value)
	return err
}

func main() {
	// Считываем аргументы командной строки
	address := flag.String("address", "127.0.0.1:8081", "адрес для запуска сервера")
	flag.Parse()

	// Формирование строки подключения для postgres
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Создание соединения с сервером postgres
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаем провайдер для БД с набором методов
	dp := DatabaseProvider{db: db}
	// Создаем экземпляр структуры с набором обработчиков
	h := Handlers{dbProvider: dp}

	// Создаем таблицы, если их нет
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS counter (
		id SERIAL PRIMARY KEY,
		value INTEGER NOT NULL
	);`)
	if err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	// Регистрируем обработчики
	http.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetCount(w, r)
		} else if r.Method == http.MethodPost {
			h.PostCount(w, r)
		} else {
			http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		}
	})

	// Запускаем веб-сервер на указанном адресе
	err = http.ListenAndServe(*address, nil)
	if err != nil {
		log.Fatal(err)
	}
}
