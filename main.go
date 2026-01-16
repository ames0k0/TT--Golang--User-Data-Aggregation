package main

import (
	"fmt"
	"log"
	"net/http"
)

// TODO (ames0k0): sId :: string -> UUID

func subscriptionsCreateHandler(w http.ResponseWriter, r *http.Request, sId string) {
	fmt.Fprintf(w, "Create, I love %s!", r.URL.Path[1:])
}

func subscriptionsReadHandler(w http.ResponseWriter, r *http.Request, sId string) {
	fmt.Fprintf(w, "Read, I love %s!", r.URL.Path[1:])
}

func subscriptionsUpdateHandler(w http.ResponseWriter, r *http.Request, sId string) {
	fmt.Fprintf(w, "Update, I love %s!", r.URL.Path[1:])
}

func subscriptionsDeleteHandler(w http.ResponseWriter, r *http.Request, sId string) {
	fmt.Fprintf(w, "Delete, I love %s!", r.URL.Path[1:])
}

func subscriptionsListHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "List, I love %s!", r.URL.Path[1:])
}

func calcSubscriptionsTotalCostsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userId := queryParams.Get("userId")
	if userId == "" {
		http.Error(w, "Name parameter is required (`userId`)", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "CalcTotalCosts, I love %s!", r.URL.Path[1:])
}

func subscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	// fmt.Println(queryParams)
	subscriptionId := queryParams.Get("subscriptionId")
	// fmt.Fprintf(w, "Read, I love %s!", r.URL.Path[1:])

	switch r.Method {
	case http.MethodPost:
		subscriptionsCreateHandler(w, r, subscriptionId)
	case http.MethodGet:
		if subscriptionId != "" {
			subscriptionsReadHandler(w, r, subscriptionId)
		} else {
			subscriptionsListHandler(w, r)
		}
	case http.MethodPatch:
		// TODO (ames0k0): Better way to handle
		if subscriptionId == "" {
			http.Error(w, "Name parameter is required (`subscriptionId`)", http.StatusBadRequest)
			return
		}
		subscriptionsUpdateHandler(w, r, subscriptionId)
	case http.MethodDelete:
		if subscriptionId == "" {
			http.Error(w, "Name parameter is required (`subscriptionId`)", http.StatusBadRequest)
			return
		}
		subscriptionsDeleteHandler(w, r, subscriptionId)
	}
}

/**
Maybe
	/subscriptions/
		{CRUD}
	/subscriptions/list
	/subscriptions/calc-total-costs ??
*/

func main() {
	http.HandleFunc("/subscriptions/", subscriptionsHandler)
	http.HandleFunc("/calc-total-costs/", calcSubscriptionsTotalCostsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
