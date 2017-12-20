/**
 * @author		John Aran (Ilyas Aranzhanovich Toxanbayev)
 * @version		1.0.0
 * @based on
 * @email      	il.aranov@gmail.com
 * @link
 * @github      https://github.com/ilyaran/github.com/ilyaran/Pump
 * @license		MIT License Copyright (c) 2017 John Aran (Ilyas Toxanbayev)
 */
package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
	"encoding/base64"
	"strings"
	"image"
	"github.com/disintegration/imaging"
	"bytes"
	"image/jpeg"
	"image/gif"
	"image/png"
	"strconv"
	"log"
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	pborman "github.com/pborman/uuid"
	"net/url"
	"time"
)
// Command-line flags.
var(
	httpAddr   = flag.String("http", ":3008", "Listen address")
 	assetsPath = "assets"
)

func main() {
	//create assets folder
	_, err := os.Stat(assetsPath)
	if os.IsNotExist(err) {
		os.Mkdir(assetsPath, os.ModePerm)
	}

	flag.Parse()
	router := mux.NewRouter()
	router.HandleFunc("/thumb/{filename}", index).Methods("GET")
	router.HandleFunc("/upload/{method}", index).Methods("POST")
	router.HandleFunc("/form", form).Methods("GET")

	log.Fatal(http.ListenAndServe(*httpAddr, router))
}

func index (w http.ResponseWriter, r *http.Request){
	t0:=time.Now()
	defer func(){
		fmt.Println("Benchmark:",time.Since(t0).Seconds(),"sec")
	}()
	uploadObj := Pump{Result:map[string]interface{}{}}
	//r.Body = http.MaxBytesReader(w, r.Body, 12345678)
	if r.ContentLength > 1024*1024*32 {
		uploadObj.Status = http.StatusNotAcceptable
		uploadObj.Result["error"] = "the input data exceeds limit"
		return
	}

	if r.Method == "POST" {
		switch mux.Vars(r)["method"] {
		case "multipart"	: 	uploadObj.upload_multipart(w,r)
		case "base64"		:
			base := r.FormValue("base64")
			if base!=""{
				uploadObj.upload_base64(base,w,r)
			}
		case "url"			:
			image_url := r.FormValue("url")
			if image_url!=""{
				uploadObj.upload_url(image_url,w,r)
			}
		case "bynary"					:
			fmt.Println(r.FormValue("blob"))
			//uploadObj.Status=http.StatusNoContent
		}
	}else if r.Method == "GET" {
		filename:=mux.Vars(r)["filename"]
		if filename != "" {
			uploadObj.thumb(assetsPath+"/"+filename,w,r)
		}
	}

	if uploadObj.Status > 0 {
		out,err := json.Marshal(uploadObj.Result)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if err!=nil{
			w.WriteHeader(500)
			w.Write([]byte("Json encode error"))
			return
		}
		w.WriteHeader(uploadObj.Status)
		w.Write(out)
	}
}

type Pump struct {
	Status int						`json:"status"`
	Result map[string]interface{}	`json:"result"`

}

func (s *Pump) upload_url(u string,w http.ResponseWriter, r *http.Request){
	//url := "http://i.imgur.com/m1UIjW1.jpg"
	image_url, err := url.ParseRequestURI(u)
	if err != nil {
		s.Status=http.StatusNotAcceptable
		s.Result["url"]="invalid"
		return
	}

	response, err := http.Get(image_url.String())
	//fmt.Printf("%T",response.Body)
	if err != nil {
		s.Status=http.StatusNotFound
		s.Result["get by url"]=err
		return
	}
	defer response.Body.Close()

	fileName,err := s.decode_image(response.Body)
	if err!=nil{
		s.Status=http.StatusInternalServerError
		s.Result[fileName]=err
		return
	}

	newFile:=s.open_file(fileName)
	defer newFile.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(newFile, response.Body)
	if err != nil {
		//log.Fatal(err)
		s.Status=http.StatusInternalServerError
		s.Result["error"]=err
		return
	}

	s.Status=http.StatusOK
	s.Result["filename"]=fileName
	s.Result["url"]=image_url.String()
}

// http://myrest.com/thumb/m1UIjW1.jpg
func (s *Pump)  thumb(f string, w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat(f)
	if os.IsNotExist(err){
		s.Status=http.StatusNotFound
		s.Result[f]=f + " is not existing file"
		return
	}

	width,height := 100,100

	// get content type
	contentType := "application/octet-stream";
	file, err := os.Open(f)
	if err != nil {
		s.Status=http.StatusInternalServerError
		s.Result["error"]=err
		//panic(err)
		return
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err1 := file.Read(buffer)
	if err1 != nil && err1 != io.EOF {
		s.Status=http.StatusInternalServerError
		s.Result["error"]=err1
		//panic(err1)
		return
	}
	contentType = http.DetectContentType(buffer[:n])
	// end get content type

	img, err2 := imaging.Open(f)
	if err2 != nil {
		s.Status=http.StatusInternalServerError
		s.Result["error"]=err
		//panic(err)
		return
	}
	var thumb image.Image = imaging.Thumbnail(img, width, height, imaging.CatmullRom)

	buffer1 := new(bytes.Buffer)
	if contentType == "image/jpeg" {
		if err := jpeg.Encode(buffer1, thumb, nil); err != nil {
			s.Status=http.StatusInternalServerError
			s.Result["error"]="unable to encode image."
			//log.Println("unable to encode image.")
			return
		}
	}
	if contentType == "image/gif" {
		if err := gif.Encode(buffer1, thumb, nil); err != nil {
			s.Status=http.StatusInternalServerError
			s.Result["error"]="unable to encode image."
			return
		}
	}
	if contentType == "image/png"{
		if err := png.Encode(buffer1, thumb); err != nil {
			s.Status=http.StatusInternalServerError
			s.Result["error"]="unable to encode image."
			return
		}
	}

	w.Header().Set("Pragma", "cache")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Content-type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer1.Bytes())))

	if _, err := w.Write(buffer1.Bytes()); err != nil {
		s.Status=http.StatusInternalServerError
		s.Result["error"]="unable to write image."
		return
	}
}

func  (s *Pump) upload_base64(img string, w http.ResponseWriter, r *http.Request){

	d1 := strings.SplitN(img, ",", 2)
	if !strings.Contains(d1[0], ";base64") {
		s.Status=http.StatusNotAcceptable
		s.Result["base64"]="invalid data"
		return
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(d1[1]))
	fileName,err := s.decode_image(reader)
	if err!=nil{
		s.Status=http.StatusInternalServerError
		s.Result[fileName]=err
		return
	}

	d2, err := base64.StdEncoding.DecodeString(d1[1])
	if err != nil {
		s.Status=http.StatusInternalServerError
		s.Result["base64 decode"]=err
		return
	}

	newFile:=s.open_file(fileName)
	defer newFile.Close()
	//***************

	_, err = newFile.Write(d2)
	if err != nil {
		s.Status=http.StatusInternalServerError
		s.Result["write file"]=err
		return
	}

	s.Status=http.StatusOK
	s.Result["filename"]=fileName
}

func  (s *Pump) upload_multipart(w http.ResponseWriter, r *http.Request){

	err := r.ParseMultipartForm(1024*1024*32)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		s.Status=http.StatusInternalServerError
		s.Result["parse multipart form"]=err
		return
	}
	//get a ref to the parsed multipart form
	m := r.MultipartForm

	//get the *fileheaders
	files := m.File["files[]"]
	filesLen:=len(files)
	done := make(chan bool, filesLen)
	saveFile:=func(done chan bool,i int){
		//for each fileheader, get a handle to the actual file
		file, err := files[i].Open()
		//fmt.Printf("%T",file)
		defer file.Close()
		if err != nil {
			s.Result["open file N"+strconv.Itoa(i)] = err
			done <- false
		}
		//create destination file making sure the path is writeable.
		filename  := files[i].Filename

		dst:=s.open_file(filename)
		defer dst.Close()
		//copy the uploaded file to the destination file
		if _, err := io.Copy(dst, file); err != nil {
			s.Result[filename+":copy file"]=err
			done <- false
		}
		s.Result[filename]="ok"
		done <- true
	}
	for i:=0;i<filesLen;i++{
		go saveFile(done,i)
	}
	for i:=0;i<filesLen;i++{
		<-done
	}

	s.Status=http.StatusOK
}



func (s *Pump) open_file(filename string)(*os.File){
	//open a file for writing
	newFile, err := os.Create(assetsPath+"/"+filename)
	if err != nil {
		s.Status=http.StatusInternalServerError
		s.Result["open new file"]=err
		return nil
	}
	return newFile
}
func (s *Pump) decode_image(body io.Reader)(string,error){
	_, formatType, err := image.Decode(body)
	if err != nil {
		return "image decode",err
	}
	//check file type
	if !(formatType == "jpg" || formatType == "jpeg" || formatType == "png" || formatType == "gif"){
		return "file type",err
	}

	return fmt.Sprintf("%s.%s",pborman.NewRandom(), formatType),nil
}


func  form(w http.ResponseWriter, r *http.Request){

	w.Write([]byte(`
<html>
	<body>

		<h2>Multipart</h2>
		<form action="/upload/multipart" method="post" enctype="multipart/form-data">
			<input type="file" name="files[]" multiple>
			<input type="submit" value="Send">
		</form>

		<h2>upload_base64</h2>
		<form action="/upload/base64" method="post" >
			<input type="text" name="base64" value="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==">
			<input type="submit" value="Send">
		</form>

		<h2>upload_url</h2>
		<form action="/upload/url" method="post" >
			<input type="text" name="url" value="http://i.imgur.com/m1UIjW1.jpg">
			<input type="submit" value="Send">
		</form>

	</body>
</html>

`))
}