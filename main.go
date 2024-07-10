package main

import (
	"bytes"
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

var AGILITY_API = "" //To be extracted from the secret file.
var BACKEND_API = "http://127.0.0.1:5000/fetch"

func login(payload *strings.Reader) LoginResponse {
	//General master login

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

func getContextID(username *string, password *string) string {
	// logging into Agility API and retrieve ContextID, which is needed to future interactions.

	var loginResponse LoginResponse

	// Master account
	var SECRET_KEYS *strings.Reader = strings.NewReader(`{
		"request": {
		"LoginID": "` + *username + `",
		"Password": "` + *password + `"
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

func clientLogin(username *string, password *string) []string {
	//Login client to confirm identity. Retreving default branch.

	var CLIENT_LOGIN_INFO *strings.Reader = strings.NewReader(`{
		"request": {
		  "LoginID": "` + *username + `",
		  "Password": "` + *password + `"
		}
	  }`)
	var loginResponse LoginResponse

	loginResponse = login(CLIENT_LOGIN_INFO)

	sessionResponse := loginResponse.Response
	var messageText string = sessionResponse.MessageText
	var contextID string = sessionResponse.SessionContextId
	var InitialBranch string = sessionResponse.InitialBranch
	if messageText != "" {
		log.Print("User:"+*username+"  Client login failed. :", messageText)
		var branchlist []string
		return branchlist
	} else {
		log.Print("User:" + *username + "  Client login successful")

		//Get available branches
		req, err := http.NewRequest("POST", AGILITY_API+"Session/BranchList", nil)

		if err != nil {
			log.Print("BranchLIst request failure:", err)
		}

		req.Header.Set("ContextId", contextID)
		req.Header.Set("Branch", InitialBranch)
		req.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(req)
		fmt.Println(response)
		//reading response
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal("BranchList read response body error:", err)
		}

		//Parsing Json
		var branchListResponse BranchListResponse
		if err := json.Unmarshal(body, &branchListResponse); err != nil {
			log.Print("BranchList download, cannot unmarshal JSON:", err)

		}
		sessionResponse := branchListResponse.Response
		var branchListResponseInner = sessionResponse.BranchListResponse
		var dsBranchListResponse = branchListResponseInner.DsBranchListResponse
		var dtBranchListResponse = dsBranchListResponse.DtBranchListResponse

		var branchlist []string = Map(dtBranchListResponse, BranchIdExtraction)

		return branchlist
	}

}

func changePriceGroup(branch *string, customerID *string, shipToSequence *string, operation *string, priceGroup *string, username *string, password *string) (string, string) {
	// method to interact with Agility API
	log.Print(*branch + " " + *priceGroup + " " + *customerID + " " + *shipToSequence)

	var requestBody *strings.Reader = strings.NewReader(`{
		"request": {
		  "CustomerID": "` + *customerID + `",
		  "ShiptoSequence": ` + *shipToSequence + `,
		  "BranchShiptoJSON": {
			"dsCustomerBranchShipto": {
			  "dtCustomerBranchShipto": [
				{
			  	"EnableDefaultCodes": true,
				"PriceGroupsAction": "` + *operation + `",
				"PriceGroups": "` + *priceGroup + `"
				}]
				
			}
		  }
		}
	  }`)

	log.Print(requestBody)

	req, err := http.NewRequest("POST", AGILITY_API+"Customer/CustomerBranchShiptoUpdate", requestBody)

	if err != nil {
		log.Print("Change request failure:", err)
	}

	var contextID string = getContextID(username, password)

	req.Header.Set("ContextId", contextID)
	req.Header.Set("Branch", *branch)
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
	sessionResponse := pricingGroupUdateResponse.Response
	fmt.Println(sessionResponse)
	var auditresult = sessionResponse.AuditResults
	var dtAuditResults = auditresult.DsAuditResults.DtAuditResults
	fmt.Println("audit result", dtAuditResults)
	var auditmessage = ""
	var auditSequence = 0
	if len(dtAuditResults) != 0 {
		var priceaudit = dtAuditResults[0]
		auditmessage = priceaudit.AuditText
		auditSequence = priceaudit.AuditSequence
	}

	var messageText = sessionResponse.MessageText
	var returnCode int = sessionResponse.ReturnCode
	messageText = messageText + auditmessage
	result := " "
	if returnCode == 0 && auditSequence != 1 {
		result = "Success"
	} else if returnCode == 1 || auditSequence == 1 {
		result = "Warning"
	} else if returnCode == 2 {
		result = "Error"
	} else {
		result = "Unknown Error"
	}
	return messageText, result
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	file, err := os.Open(pwd + "/secret.txt")
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	file, err := os.Open(pwd + "/secret.txt")
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
	AGILITY_API = secrets[2]
	indexDelivery := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	}
	loginRequest := func(w http.ResponseWriter, r *http.Request) {
		//extract data
		client_username := r.PostFormValue("client-username")
		client_password := r.PostFormValue("client-password")

		log.Println(client_username)
		branchList := clientLogin(&client_username, &client_password)
		if len(branchList) == 0 {
			htmlStr := fmt.Sprintf(`<p style='color:red;visibility:visible !important'>LOGIN FAILED, CHECK USERNAME OR PASSWORD</p>
					<script>
			   		document.getElementById("priceGroupForm").removeAttribute("hidden");
					</script>`)
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)

		} else {
			htmlStr1 := `<form hx-post="/add-price-group/" hx-target="#requestList" hx-swap="beforebegin"
                        hx-indicator="#spinner" id="priceGroupForm"> 
						<div class='mb-2'>
			htmlStr1 := `<form hx-post="/add-price-group/" hx-target="#requestList" hx-swap="beforebegin"
                        hx-indicator="#spinner" id="priceGroupForm"> 
						<div class='mb-2'>
			<label for='branch'>Branch</label> 
			<select name="branch" id="branch" class='form-control' required>`
			htmlStr3 := `</select>
									</div>
									<script>
									document.getElementById("LoginForm").setAttribute('hidden','');
									</script>
							<div class="mb-2">
                            <label for="operation">Operation </label>
                            <select type="text" name="operation" id="operation" class="form-control"
                                onchange="UpdateButton()" required>
                                <option value="Add" selected>Add</option>
                                <option value="Delete">Delete</option>
                            </select>
                        </div>
                        <div class="mb-2">
                            <label for="pricing-group-id">Pricing group ID</label>
                            <input type="text" name="pricing-group-id" id="pricing-group-id" class="form-control"
                                style="width:270px" required/>
                        </div>
                        <div class="mb-3">
                            <label for="customer-id">Customer ID</label>
                            <input type="text" name="customer-id" id="customer-id" class="form-control"
                                style="width:270px" required/>
                        </div>
                        <div class="mb-2">
                            <label for="customer-ship-to">Customer Ship-to</label>
                            <input type="number" name="customer-ship-to" id="customer-ship-to" class="form-control"
                                style="width:90px" required/>
                        </div>
                        <button type="submit" class="btn btn-success" id="submission-button-style">
                            <span class="spinner-border spinner-border-sm htmx-indicator" id="spinner" role="status"
                                aria-hidden="true"></span>
                            <span id="submission-button"> Add &nbsp &nbsp &nbsp</span>
                        </button> </form>`
			htmlStr2 := ""
			for _, element := range branchList {
				htmlStr2 = htmlStr2 + fmt.Sprintf(`<option value="%s">%s</option>`, element, element)
			}
			htmlStr4 := `
					<hr/>
					<h4> See what customers are in your pricing group:</h4>
					<p> Results are updated every hour.</p>
                    <form hx-post= "/fetch/" hx-target="#fetch_result"
                    hx-indicator="#spinner"> 
                    <div class="mb-3">
                        <label for="branch">Branch</label>
                        <select name="branch" id="branch" class='form-control' required>`

			htmlStr5 := `</select>
                    </div>
                    <div class="mb-2">
                        <label for="pricing-group">Pricing group </label>
                        <input name="pricing-group" id="pricing-group" class="form-control"
                            style="width:200 px" required/>
                    </div>
                    <button type="submit" class="btn btn-secondary" id="submission-button-style">
                        <span class="spinner-border spinner-border-sm htmx-indicator" id="spinner" role="status"
                            aria-hidden="true"></span>
                        <span id="submission-button"> Search &nbsp &nbsp &nbsp</span>
                    </button>
                    </form>`
			htmlStr := htmlStr1 + htmlStr2 + htmlStr3 + htmlStr4 + htmlStr2 + htmlStr5
			tmpl, _ := template.New("t").Parse(htmlStr)
			tmpl.Execute(w, nil)
		}

	}
	// handler function #2 - returns the template block with the newly added customers, as an HTMX response
	operationRequest := func(w http.ResponseWriter, r *http.Request) {

		//delaying send request
		milliseconds := 400
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)

		//extract data
		branch := r.PostFormValue("branch")
		operation := r.PostFormValue("operation")
		CustomerID := r.PostFormValue("customer-id")
		ShiptoSequence := r.PostFormValue("customer-ship-to")
		priceGroup := r.PostFormValue("pricing-group-id")

		messageText, result := changePriceGroup(&branch, &CustomerID, &ShiptoSequence, &operation, &priceGroup, &USERNAME, &PASSWORD)
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
	fetch := func(w http.ResponseWriter, r *http.Request) {

		branch := r.PostFormValue("branch")
		priceGroup := r.PostFormValue("pricing-group")

		post_body := []byte(`{
			"branch":"` + branch + `",
			"pricing_group":"` + priceGroup + `"}`)
		// var requestBody *strings.Reader = strings.NewReader(`{
		// 	"branch":` + branch + `
		// 	"pricing_group:"` + priceGroup + `
		//   }`)
		req, _ := http.NewRequest("POST", BACKEND_API, bytes.NewBuffer(post_body))

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("method", "post")
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Print("Fetch pricing group error: ", err)
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal("Backend reply parsing error:", err)
		}

		//Parsing Json
		var pricingGroupList PricingGroupList
		if err := json.Unmarshal(body, &pricingGroupList); err != nil {
			log.Print("Backend reply , cannot unmarshal JSON:", err)
		}

		var priceGrouplist = pricingGroupList.Response

		htmlStr1 := `<table>
						<tr>
							<th>Customer Name</th>
							<th>Customer ID</th>
							<th>Ship-to</th>
						</tr>`
		htmlStr2 := ``
		for _, element := range priceGrouplist {
			htmlStr2 = htmlStr2 + fmt.Sprintf(`<tr><td>%s</td>
											<td>%s</td>
											<td>%v</td>
											</tr>`, element.Name, element.ID, element.Shipto)
		}

		htmlStr3 := `</table>`
		tmpl, _ := template.New("t").Parse(htmlStr1 + htmlStr2 + htmlStr3)
		tmpl.Execute(w, nil)
	}
	// log.Print(clientLogin("e135478", "Changeme123"))
	// ID: COLLUPA S.T.: 2

	http.HandleFunc("/", indexDelivery)
	http.HandleFunc("/login/", loginRequest)
	http.HandleFunc("/add-price-group/", operationRequest)
	http.HandleFunc("/fetch/", fetch)
	log.Fatal(http.ListenAndServe(":8045", nil))
}
