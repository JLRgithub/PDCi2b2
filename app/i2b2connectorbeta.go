package main

import(
	"net/http"
	"io/ioutil"
	"gopkg.in/dedis/onet.v1/log"
	"github.com/gorilla/mux"
	"time"
	"fmt"
	"encoding/json"
	"gopkg.in/urfave/cli.v1"
	"strconv"
	"github.com/JLRgithub/PrivateDCi2b2/services"
	"gopkg.in/dedis/onet.v1"
)

type State struct{
	address string
	group string
}

func webServer(c *cli.Context){
	r := mux.NewRouter()
	address := ""
	if a := c.String("address"); a != "" {
		address = a
	}
	group := ""
	if g := c.String("file"); g != "" {
		group = g
	}

	s := State{address,group}
	
	r.HandleFunc("/totalnum", s.totalNumHandler)
	r.HandleFunc("/totalnums", s.totalNumsHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

type Result struct{
	ConceptPath string `json:"conceptpath"`
	TotalNum string `json:"totalnum"`
}
type ResultGroup struct{
	Group string `json:"group"`
	TotalNum string `json:"totalnum"`
}
type Response struct{
	Concepts []Result `json:"concepts"`
}
type ResponseGroup struct{
	Groups []ResultGroup `json:"groups"`
}
/*
This handler finds the total num for each of the concept paths given as input, no group by location nor time is performed.
The results are aggregated to return 1 encrypted total num per concept path.
*/
func (state State) totalNumHandler(w http.ResponseWriter, r *http.Request) {
	start0 := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	body, err1 := ioutil.ReadAll(r.Body)
	if(err1!=nil){
		log.Fatal("Could not read request body.", err1)
	}
	var pathsObj map[string]interface{}
	json.Unmarshal(body,&pathsObj)
	paths := pathsObj["conceptpaths"].([]interface{})
	keyString := pathsObj["clientpublickey"].(string)
	fmt.Println("received ",len(paths)," paths")
	fmt.Println("totalNum received public key : ",keyString)
	if(len(paths)>0){

		el, err := openGroupToml(state.group)
		if err != nil {
			log.Fatal("Could not open group toml file.",err)
		}

		

		var results []Result
		resultsChannel := make(chan []Result, len(paths))
		for i,path := range paths{
			client := serviceI2B2dc.NewClientFromKey(el.List[0], strconv.Itoa(i), keyString, false)
			go queryAggr(resultsChannel ,path.(string),client,el)
		}
		for range paths{
			queryres := <-resultsChannel
			if(queryres!=nil){
				results = append(results, queryres...)
			}
		}
		
		res := &Response{results}
		fmt.Println("response : ",*res)
		resJson,_ := json.Marshal(res)
		w.Write(resJson)
	}else{
		res := &Response{nil}
		resJson,_ := json.Marshal(res)
		w.Write(resJson)
	}
	fmt.Println("Handler time: ", time.Since(start0))
}
/*
This handler finds the total nums for a given concept path according to location and time. It returns 2 lists of encrypted
total nums, 1 where the results are grouped by location, the other by time.
*/
func (state State) totalNumsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	body, err1 := ioutil.ReadAll(r.Body)
	if(err1!=nil){
		log.Fatal("Could not read request body.", err1)
	}
	var pathObj map[string]interface{}
	json.Unmarshal(body,&pathObj)
	path := pathObj["conceptpath"].(string)
	keyString := pathObj["clientpublickey"].(string)
	from := pathObj["fromtime"].(string)
	to := pathObj["totime"].(string)
	distribution := pathObj["distribution"].(string)
	fmt.Println("totalNums received path : ",path)
	fmt.Println("and public key : ",keyString)
	fmt.Println("with time frame : ",from,"-",to)
	fmt.Println(distribution+" distribution")

	el, err := openGroupToml(state.group)
	if err != nil {
		log.Fatal("Could not open group toml file.",err)
	}
	client := serviceI2B2dc.NewClientFromKey(el.List[0], strconv.Itoa(0), keyString, false)

	var groupBy []string
	groupBy = append(groupBy, "location_cd")
	if(distribution=="point"){//otherwise it is "cumulative". this way we return either the evolution over time or the aggregation.
		groupBy = append(groupBy, "year")
	}
	
	results := queryGroupBy(path,client,el,from,to, groupBy)
	res := &ResponseGroup{results}
	resJson,_ := json.Marshal(res)
	w.Write(resJson)
}

func queryAggr(resch chan []Result, path string, client  *serviceI2B2dc.APIremote, el *onet.Roster) []Result{
	queryID, err := client.SendQuery(el, serviceI2B2dc.QueryID(""), nil, []string{},
		[]string{}, []string{path}, []string{},"","")
	if err != nil {
		fmt.Println("Service did not start.", err)
		return nil
	}
	fmt.Println("at time : ", time.Now())
	grps, aggr, err := client.ExecuteQuery(*queryID)
	if err != nil {
		fmt.Println("Query could not be executed.", err)
		return nil
	}
	var results []Result
	if(grps!=nil && aggr!=nil){
		for i := 0; i < len(*grps); i++ {
			r := Result{path, (*aggr)[i]}
			fmt.Println(r)
			results = append(results, r)
		}
	}else{
		results = append(results, Result{path, (*aggr)[0]})
	}
	resch <- results
	return results
}

func queryGroupBy(path string, client  *serviceI2B2dc.APIremote, el *onet.Roster, from string, to string, groupBy []string) []ResultGroup{
	queryID, err := client.SendQuery(el, serviceI2B2dc.QueryID(""), nil, []string{}, []string{}, []string{path}, groupBy, from, to)
	if err != nil {
		fmt.Println("Service did not start.", err)
		return nil
	}

	grps, aggr, err := client.ExecuteQuery(*queryID)
	if err != nil {
		fmt.Println("Query could not be executed.", err)
		return nil
	}
	var results []ResultGroup
	if(grps!=nil && aggr!=nil){
		for i := 0; i < len(*grps); i++ {
			r := ResultGroup{(*grps)[i], (*aggr)[i]}
			fmt.Println(r)
			results = append(results, r)
		}
	}
	return results
}