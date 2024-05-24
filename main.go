package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Utility Map function for mapping arrays
func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}
func BranchIdExtraction(x BranchListInner) string {
	return x.BranchID
}

const AGILITY_API = "https://agilitywmstest.bc.com/Prod_DB_IIAgilityPublic/rest/"

func login(payload *strings.Reader) LoginResponse {
	response, err := http.Post(AGILITY_API+"Session/Login", "application/json", payload)
	if err != nil {
		log.Fatal("Master login error: ", err)
	}
	defer response.Body.Close()

	//reading response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	//Parsing Json
	var loginResponse LoginResponse
	if err := json.Unmarshal(body, &loginResponse); err != nil {
		log.Print("Master login, cannot unmarshal JSON:", err)

	}
	return loginResponse
}

func getContextID(username string, password string) string {
	// logging into Agility API and retrieve ContextID, which is needed to future interactions.

	var loginResponse LoginResponse

	// Master account
	var SECRET_KEYS *strings.Reader = strings.NewReader(`{
		"request": {
		"LoginID": "` + username + `",
		"Password": "` + password + `"
		}
	}`)

	loginResponse = login(SECRET_KEYS)

	sessionResponse := loginResponse.Response

	var contextID string = sessionResponse.SessionContextId
	var messageText string = sessionResponse.MessageText

	if messageText != "" {
		log.Print("Master login failed. :", messageText)
	} else {
		log.Print("Master login successful.")
	}

	return contextID
}

func clientLogin(username string, password string) []string {
	//Login client to confirm identity. Retreving default branch.

	var CLIENT_LOGIN_INFO *strings.Reader = strings.NewReader(`{
		"request": {
		  "LoginID": "` + username + `",
		  "Password": "` + password + `"
		}
	  }`)
	var loginResponse LoginResponse

	loginResponse = login(CLIENT_LOGIN_INFO)

	sessionResponse := loginResponse.Response
	var messageText string = sessionResponse.MessageText
	var contextID string = sessionResponse.SessionContextId
	var InitialBranch string = sessionResponse.InitialBranch
	if messageText != "" {
		log.Print("User:"+username+"  Client login failed. :", messageText)
		var branchlist []string
		return branchlist
	} else {
		log.Print("User:" + username + "  Client login successful.")

		//Get available branches
		req, err := http.NewRequest("POST", AGILITY_API+"Session/BranchList", nil)

		if err != nil {
			log.Print("BranchLIst request failure:", err)
		}

		req.Header.Set("ContextId", contextID)
		req.Header.Set("Branch", InitialBranch)
		req.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(req)

		//reading response
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal("BranchList read response body error:", err)
		}

		//Parsing Json
		var branchListResponse BranchListResponse
		if err := json.Unmarshal(body, &branchListResponse); err != nil {
			log.Print("Pricing Group Update , cannot unmarshal JSON:", err)

		}
		sessionResponse := branchListResponse.Response
		var branchListResponseInner = sessionResponse.BranchListResponse
		var dsBranchListResponse = branchListResponseInner.DsBranchListResponse
		var dtBranchListResponse = dsBranchListResponse.DtBranchListResponse

		var branchlist []string = Map(dtBranchListResponse, BranchIdExtraction)

		return branchlist
	}

}

func changePriceGroup(contextID string, branch string, customerID string, shipToSequence string, operation string, priceGroup string) (string, string) {
	// method to interact with Agility API

	var requestBody *strings.Reader = strings.NewReader(`{
		"request": {
		  "CustomerID": "` + customerID + `",
		  "ShiptoSequence": "` + shipToSequence + `",
		  "BranchShiptoJSON": {
			"dsCustomerBranchShipto": {
			  "dtCustomerBranchShipto": [
				{"PriceGroupsAction": "` + operation + `",
				"PriceGroups": "` + priceGroup + `"}]
			}
		  }
		}
	  }`)
	req, err := http.NewRequest("POST", AGILITY_API+"Customer/CustomerBranchShiptoUpdate", requestBody)

	if err != nil {
		log.Print("Change request failure:", err)
	}

	req.Header.Set("ContextId", contextID)
	req.Header.Set("Branch", branch)
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)

	//reading response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Pricing Group Update read response body error:", err)
	}

	//Parsing Json
	var pricingGroupUdateResponse PricingGroupUpdateResponse
	if err := json.Unmarshal(body, &pricingGroupUdateResponse); err != nil {
		log.Print("Pricing Group Update , cannot unmarshal JSON:", err)

	}

	// fmt.Println("Price Group response:", pricingGroupUdateResponse)

	sessionResponse := pricingGroupUdateResponse.Response

	var messageText = sessionResponse.MessageText
	var returnCode int = sessionResponse.ReturnCode
	result := " "
	if returnCode == 0 {
		result = "Success"
	} else if returnCode == 1 {
		result = "Warning"
	} else {
		result = "Error"
	}
	return messageText, result
}

func main() {
	file, err := os.Open("secret.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	secrets := strings.Fields(string(data))
	USERNAME := secrets[0]
	PASSWORD := secrets[1]

	var contextID string = getContextID(USERNAME, PASSWORD)

	main := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	}
	loginRequest := func(w http.ResponseWriter, r *http.Request) {
		//extract data
		client_username := r.PostFormValue("client-username")
		client_password := r.PostFormValue("client-password")
		branchList := clientLogin(client_username, client_password)
		if len(branchList) == 0 {
			htmlStr := fmt.Sprintf(`<p style='color:red;visibility:visible !important' >LOGIN FAILED, CHECK USERNAME OR PASSWORD</p>
					<script>
			   		document.getElementById("priceGroupForm").removeAttribute("hidden");
					</script>`)
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)

		} else {
			htmlStr1 := `<div class='mb-2'>
			<label for='branch'>Branch</label> 
			<select name="branch" id="branch" class='form-control' required>`
			htmlStr3 := `</select> 
									<script>
									document.getElementById("LoginForm").setAttribute('hidden','');
           							document.getElementById("priceGroupForm").style.visibility='visible';
									</script>`
			htmlStr2 := ""
			for _, element := range branchList {
				htmlStr2 = htmlStr2 + fmt.Sprintf(`<option value="%s">%s</option>`, element, element)
			}

			htmlStr := htmlStr1 + htmlStr2 + htmlStr3
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)
		}

	}
	// handler function #2 - returns the template block with the newly added customers, as an HTMX response
	addRequest := func(w http.ResponseWriter, r *http.Request) {

		//delaying send request
		milliseconds := 400
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)

		//extract data
		branch := r.PostFormValue("branch")
		operation := r.PostFormValue("operation")
		CustomerID := r.PostFormValue("customer-id")
		ShiptoSequence := r.PostFormValue("customer-ship-to")
		priceGroup := r.PostFormValue("pricing-group-id")

		log.Print(branch + " " + priceGroup + " " + CustomerID + " " + ShiptoSequence)

		messageText, result := changePriceGroup(contextID, branch, CustomerID, ShiptoSequence, operation, priceGroup)
		htmlStr := ""
		if result == "Success" {
			htmlStr = fmt.Sprintf(`<tr><th scope="row">%s</th>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td style='color:green'>%s</td>
			<td>%s</td>
		</tr>`, operation, priceGroup, CustomerID, ShiptoSequence, result, messageText)
		} else if result == "Warning" {
			htmlStr = fmt.Sprintf(`<tr><th scope="row">%s</th>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td style='color:orange'>%s</td>
			<td>%s</td>
		</tr>`, operation, priceGroup, CustomerID, ShiptoSequence, result, messageText)
		} else {
			htmlStr = fmt.Sprintf(`<tr><th scope="row">%s</th>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td style='color:red'>%s</td>
			<td>%s</td>
		</tr>`, operation, priceGroup, CustomerID, ShiptoSequence, result, messageText)
		}

		tmpl, _ := template.New("t").Parse(htmlStr)
		tmpl.Execute(w, nil)

		// tmpl := template.Must(template.ParseFiles("index.html"))
		// tmpl.Execute(w, map[string]interface{}{"message": "message"})
	}

	// log.Print(clientLogin("e135478", "Changeme123"))
	// ID: COLLUPA S.T.: 2

	http.HandleFunc("/", main)
	http.HandleFunc("/login/", loginRequest)
	http.HandleFunc("/add-price-group/", addRequest)

	log.Fatal(http.ListenAndServe(":8045", nil))
}
