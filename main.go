package main

import (
	"context"
	"errors"

	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// XXX (ames0k0): Application Struct ?
type application struct {
	logger *slog.Logger
	dbpool *pgxpool.Pool
}

type UserSubscriptions struct {
	Id pgtype.UUID `json:"id"`
	User_id pgtype.UUID `json:"user_id"`
	Service_name string `json:"service_name"`
	Price int `json:"price"`
	Start_date string `json:"start_date"`
	End_date *string `json:"end_date"`
}

type UserSubscriptionsTotalCost struct {
	ServicesCount int `json:"services_count"`
	ServicesTotalPrice int `json:"services_total_price"`
}

// TODO (ames0k0): sId :: string -> UUID
func main() {
	// LOGGING init
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// DATABASE connection
	// dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	dbpool, err := pgxpool.New(context.Background(), "postgresql://postgres:simple@localhost:5454/templates__postgres")
	if err != nil {
		logger.Error(
			"Unable to create pg connection pool",
			"err",
			err.Error(),
		)
		os.Exit(1)
	}
	logger.Info("Connected to the database.")
	defer dbpool.Close()

	app := &application{logger: logger, dbpool: dbpool}

	http.HandleFunc("/subscriptions/calc-total-cost/", app.subscriptionsCalcTotalCostHandler)
	http.HandleFunc("/subscriptions/list/", app.subscriptionsListHandler)
	http.HandleFunc("/subscriptions/", app.subscriptionsHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
		app.logger.Error(
			"Listen and Server FAILED",
			"err",
			err.Error(),
		)
	}
}

func (app *application) subscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	var user_subscription UserSubscriptions

	// XXX (ames0k0): Data Preload (validation)
	if (r.Method != http.MethodPost) {
		requiredQP := []string{"id"}
		optionalQP := []string{}
		rRQP, _, err := queryParamsLoader(w, r, app, requiredQP, optionalQP)

		if err != nil {
			return
		}

		query := `
		SELECT * FROM user_subscriptions
		WHERE id = $1
		`

		err = app.dbpool.QueryRow(
			context.Background(),
			query,
			rRQP["id"],
		).Scan(
			&user_subscription.Id,
			&user_subscription.User_id,
			&user_subscription.Service_name,
			&user_subscription.Price,
			&user_subscription.Start_date,
			&user_subscription.End_date,
		)

		if err != nil {
			http.Error(
				w,
				"UserSubscriptions is not found!",
				http.StatusNotFound,
			)
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		app.subscriptionsCreateHandler(w, r)
	case http.MethodGet:
		app.subscriptionsReadHandler(w, r, user_subscription)
	case http.MethodPatch:
		app.subscriptionsUpdateHandler(w, r, user_subscription)
	case http.MethodDelete:
		app.subscriptionsDeleteHandler(w, r, user_subscription)
	}
}

func queryParamsLoader(
	w http.ResponseWriter, r *http.Request,
	app *application,
	requiredQP []string, optionalQP[]string,
) (
	map[string]string, map[string]string,
	error,
) {
	rRQP := make(map[string]string)
	rOQP := make(map[string]string)
	var rMRQP []string

	queryParams := r.URL.Query()

	for _, key := range requiredQP {
		value := strings.TrimSpace(queryParams.Get(key))
		if value == "" {
			rMRQP = append(rMRQP, key)
		} else {
			rRQP[key] = value
		}
	}

	for _, key := range optionalQP {
		value := strings.TrimSpace(queryParams.Get(key))
		rOQP[key] = value
	}

	if len(rMRQP) > 0 {
		errMsg := "Missing required params: " + strings.Join(rMRQP, ", ")
		http.Error(
			w,
			errMsg,
			http.StatusBadRequest,
		)
		app.logger.Error(
			errMsg,
			"request.Form",
			r.Form,
		)
		return rRQP, rOQP, errors.New(errMsg)
	}

	return rRQP, rOQP, nil
}


func formValuesLoader(
	w http.ResponseWriter, r *http.Request,
	app *application,
	requiredFV []string, optionalFV[]string,
) (
	map[string]string,
	map[string]interface{},
	error,
) {
	rRFV := make(map[string]string)
	rOFV := make(map[string]interface{})
	var rMRFV []string

	err := r.ParseForm()
	if err != nil {
		errMsg := "Could not parse request.Form: " + r.Method
		http.Error(
			w,
			errMsg,
			http.StatusBadRequest,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"request.Form",
			r.Form,
		)
		return rRFV, rOFV, errors.New(errMsg)
	}

	for _, key := range requiredFV {
		value := strings.TrimSpace(r.FormValue(key))
		if value == "" {
			rMRFV = append(rMRFV, key)
		} else {
			rRFV[key] = value
		}
	}

	if len(rMRFV) > 0 {
		errMsg := "Missing required form data: " + strings.Join(rMRFV, ", ")
		http.Error(
			w,
			errMsg,
			http.StatusBadRequest,
		)
		app.logger.Error(
			errMsg,
			"request.form",
			r.Form,
		)
		return rRFV, rOFV, errors.New(errMsg)
	}

	for _, key := range optionalFV {
		value := strings.TrimSpace(r.FormValue(key))
		if value == "" {
			rOFV[key] = sql.NullString{
				String: r.FormValue("end_date"), Valid: false,
			}
		} else {
			rOFV[key] = sql.NullString{
				String: r.FormValue("end_date"), Valid: true,
			}
		}
	}

	return rRFV, rOFV, nil
}

func (app *application) subscriptionsCreateHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		errMsg := "Could not parse request.Form"
		http.Error(
			w,
			errMsg,
			http.StatusBadRequest,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"request.Form",
			r.Form,
		)
		return
	}

	requiredFD := []string{"user_id", "service_name", "price", "start_date"}
	var rMRFD[]string

	for _, key := range requiredFD {
		value := r.FormValue(key)
		if value == "" {
			rMRFD = append(rMRFD, key)
		}
	}

	if len(rMRFD) > 0 {
		errMsg := "Missing required form data: " + strings.Join(rMRFD, ", ")
		http.Error(
			w,
			errMsg,
			http.StatusBadRequest,
		)
		app.logger.Error(
			errMsg,
			"request.form",
			r.Form,
		)
		return
	}

	var end_date sql.NullString

	if r.FormValue("end_date") == "" {
		end_date = sql.NullString{String: r.FormValue("end_date"), Valid: false}
	} else {
		end_date = sql.NullString{String: r.FormValue("end_date"), Valid: true}
	}

	query := `
	INSERT INTO user_subscriptions (user_id, service_name, price, start_date, end_date)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err = app.dbpool.Exec(
		context.Background(),
		query,
		r.FormValue("user_id"),
		r.FormValue("service_name"),
		r.FormValue("price"),
		r.FormValue("start_date"),
		end_date,
	)

	if err != nil {
		errMsg := "Could not dbpool.Exec(Insert)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"request.Form",
			r.Form,
		)
		return
	}
}

func (app *application) subscriptionsReadHandler(w http.ResponseWriter, _ *http.Request, user_subscription UserSubscriptions) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(user_subscription)
	if err != nil {
		errMsg := "Could not json.encode(UserSubscriptions)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"user_subscription",
			user_subscription,
		)
		return
	}
}

func (app *application) subscriptionsUpdateHandler(
	w http.ResponseWriter, r *http.Request,
	user_subscription UserSubscriptions,
) {
	requiredFV := []string{"user_id", "service_name", "price", "start_date"}
	optionalFV := []string{"end_date"}

	rRFV, rOFV, err := formValuesLoader(
		w, r, app, requiredFV, optionalFV,
	)

	if err != nil {
		return
	}

	query := `
	UPDATE user_subscriptions
	SET
		user_id = $2,
		service_name = $3,
		price = $4,
		start_date = $5,
		end_date = $6
	WHERE
		id = $1
	`

	_, err = app.dbpool.Exec(
		context.Background(),
		query,
		user_subscription.Id,
		rRFV["user_id"],
		rRFV["service_name"],
		rRFV["price"],
		rRFV["start_date"],
		rOFV["end_date"],
	)

	if err != nil {
		errMsg := "Could not dbpool.Exec(Update)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"request.Form",
			r.Form,
		)
		return
	}
}


func (app *application) subscriptionsDeleteHandler(
	w http.ResponseWriter, _ *http.Request,
	user_subscription UserSubscriptions,
) {
	query := `DELETE FROM user_subscriptions WHERE id = $1`

	_, err := app.dbpool.Exec(
		context.Background(),
		query,
		user_subscription.Id,
	)

	if err != nil {
		errMsg := "Could not dbpool.Exec(Delete)"
		http.Error(w, errMsg, http.StatusInternalServerError)
		app.logger.Error(errMsg, "err", err.Error())
		return
	}
}

func (app *application) subscriptionsListHandler(w http.ResponseWriter, r *http.Request) {
	result_limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	var users_subscriptions []UserSubscriptions

	if err != nil || result_limit > 50 {
		result_limit = 50
	}

	rows, err := app.dbpool.Query(
		context.Background(),
		"SELECT * FROM user_subscriptions LIMIT $1",
		result_limit,
	)
	if err != nil {
		errMsg := "Could select data from table 'user_subscriptions'"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
		)
	}
	defer rows.Close()

	for rows.Next() {
		var user_subscription UserSubscriptions
		err := rows.Scan(
			&user_subscription.Id,
			&user_subscription.User_id,
			&user_subscription.Service_name,
			&user_subscription.Price,
			&user_subscription.Start_date,
			&user_subscription.End_date,
		)
		if err != nil {
			errMsg := "Could not dbpool.Query(List.Load)"
			http.Error(
				w,
				errMsg,
				http.StatusInternalServerError,
			)
			app.logger.Error(
				errMsg,
				"err",
				err.Error(),
			)
		}
		users_subscriptions = append(users_subscriptions, user_subscription)
	}

	err = rows.Err()
	if err != nil {
		errMsg := "Could not dbpool.Query(Rows)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(users_subscriptions)

	if err != nil {
		errMsg := "Could not json.encode([]UserSubscriptions)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"users_subscriptions",
			users_subscriptions,
		)
		return
	}
}

func (app *application) subscriptionsCalcTotalCostHandler(w http.ResponseWriter, r *http.Request) {
	requiredQP := []string{"user_id", "service_name", "start_date", "end_date"}
	optionalQP := []string{}
	rRQP, _, err := queryParamsLoader(w, r, app, requiredQP, optionalQP)

	if err != nil {
		return
	}

	var user_subscriptions_total_cost UserSubscriptionsTotalCost

	query := `
	SELECT
		count(id), sum(price) FROM user_subscriptions
	WHERE
		user_id = $1 AND
		service_name = $2 AND
		start_date = $3 AND
		(
			end_date IS NULL OR end_date <= $4
		)
	`

	err = app.dbpool.QueryRow(
		context.Background(),
		query,
		rRQP["user_id"],
		rRQP["service_name"],
		rRQP["start_date"],
		rRQP["end_date"],
	).Scan(
		&user_subscriptions_total_cost.ServicesCount,
		&user_subscriptions_total_cost.ServicesTotalPrice,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(user_subscriptions_total_cost)

	if err != nil {
		errMsg := "Could not json.encode([]UserSubscriptionsTotalCost)"
		http.Error(
			w,
			errMsg,
			http.StatusInternalServerError,
		)
		app.logger.Error(
			errMsg,
			"err",
			err.Error(),
			"user_subscriptions_total_cost",
			user_subscriptions_total_cost,
		)
		return
	}
}
