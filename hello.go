package hello

import (
    "fmt"
    "html/template"
    "net/http"
    "encoding/json"
    "appengine"
    "appengine/urlfetch"
    "log"
    "errors"
    "strings"
)

func init() {
    http.HandleFunc("/", root)
    http.HandleFunc("/temperature", temperature)
}

func root(w http.ResponseWriter, r *http.Request) {

    fmt.Fprint(w, inputForm)
}

const inputForm = `
<html>
  <body>
    Enter names of cities, maybe 5 (separated by commas):
    <form action="/temperature" method="post">
      <div><textarea name="content" rows="3" cols="60"></textarea></div>
      <div><input type="submit" value="Submit"></div>
    </form>
  </body>
</html>
`

func temperature(w http.ResponseWriter, r *http.Request) {

    

    weatherReports := make(chan PrettyWeather)
    errs := make(chan error)

    cities := strings.Split(r.FormValue("content"), ",")


    // go routine with an anon function
    for _, city := range cities {
       go func(c string) {



            data, fetcherr := query(strings.TrimSpace(c), r)


            if fetcherr != nil {
                errs <- fetcherr
                return
            }


            var weather PrettyWeather

            weather.Maximum = data.Data.Weather[0].Max
            weather.Minimum = data.Data.Weather[0].Min
            weather.Name = data.Data.Request[0].City


            weatherReports <- weather
        }(city)


    }

    var reports []PrettyWeather



    // Collect everything with select.
    for i := 0; i < len(cities); i++ {
        select {
        case temp := <-weatherReports:

            reports = append(reports, temp)


        case err := <-errs:
            log.Print("error")
            log.Print(err)

        }
    }


    err := outputTemplate.Execute(w,reports)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func query(city string, r *http.Request) (weatherData, error) {
	c := appengine.NewContext(r)
    client := urlfetch.Client(c)




    resp, err := client.Get("http://api.worldweatheronline.com/free/v1/weather.ashx?key=xxxxxxxxx&format=json&q=" + city)
    if err != nil {
        return weatherData{}, err
    }

    defer resp.Body.Close()

    var d weatherData


    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return weatherData{}, err
    }

    if (len(d.Data.ErrorM) > 0) {
        return weatherData{}, errors.New(d.Data.ErrorM[0].Message)
    }


    return d, nil
}




type wwData struct {
    Weather []weather `json:"weather"`
    Request []request `json:"request"`
    ErrorM []errorm `json:"error"`


}

type weather struct {
    Max string `json:"tempMaxC"`
    Min string `json:"tempMinC"`


}

type request struct {
    City string `json:"query"`

}


type errorm struct {
    Message string `json:"msg"`

}


type weatherData struct {
    Data wwData `json:"data"`
}


type PrettyWeather struct {
    Maximum string
    Minimum string
    Name string
}





var outputTemplate = template.Must(template.New("temperature").Parse(outputTemplateHTML))

const outputTemplateHTML = `
<html>
  <body>

    {{ $cities := . }} 
    {{range $index, $element := $cities }}
       <li><strong>{{$element.Name}}</strong>: Maximum: {{$element.Maximum}} C, Minimum: {{$element.Minimum}} C </li>
    {{end}}


  </body>
</html>
`