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
	"time"
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
)
// Command-line flags.
var (
	httpAddr   = flag.String("http", ":3008", "Listen address")
	pollPeriod = flag.Duration("poll", 5*time.Second, "Poll period")
	version    = flag.String("version", "1.4", "Go version")
)
func main() {
	flag.Parse()
	http.HandleFunc("/", index)
	http.HandleFunc("/form", form)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

type Pump struct {
	Status int
	Result map[string]interface{}
}

func index (w http.ResponseWriter, r *http.Request){
	upload := Pump{Result:map[string]interface{}{}}
	if r.Method == "GET" {
		f := r.URL.Query().Get("f")
		if f != "" {
			upload.thumb(f,w,r)
		}
	}else if r.Method == "POST" {
		url := r.FormValue("url")
		if url != "" {
			upload.upload_url(url,w,r)
		}else {
			img := r.FormValue("image")
			if img != "" {
				upload.upload_base64(img, w, r)
			}else {
				upload.upload(w,r)
			}
		}
	}

	if upload.Status > 0 {
		out,_:=json.Marshal(upload)
		w.WriteHeader(upload.Status)
		w.Write(out)
	}

}
func (s *Pump) open_file(filename string)(*os.File){
	//open a file for writing
	file, err := os.Create(filename)
	if err != nil {
		s.Status=500
		s.Result["open_file"]=err
		log.Fatal(err)
		return nil
	}
	return file
}
func (s *Pump) upload_url(url string,w http.ResponseWriter, r *http.Request){
	//url := "http://i.imgur.com/m1UIjW1.jpg"
	// don't worry about errors
	response, err := http.Get(url)
	if err != nil {
		s.Status=500
		s.Result["error"]=err
		return
		//log.Fatal(e)
	}

	defer response.Body.Close()

	file:=s.open_file("asdf.jpg")
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		//log.Fatal(err)
		s.Status=500
		s.Result["error"]=err
		return
	}
	//file.Close()
	//fmt.Println("Success!")
	s.Status=200
	s.Result["filename"]="asdf.jpg"
	s.Result["url"]=url
}
func  (s *Pump) thumb(f string,w http.ResponseWriter, r *http.Request) {

	_, err := os.Stat(f)
	if os.IsNotExist(err){
		//fmt.Fprintf(w, f + " is not exists file")
		s.Status=404
		s.Result[f]=f + " is not exists file"
		return
	}

	width,height := 100,100

	// get content type
	contentType := "application/octet-stream";
	file, err := os.Open(f)
	if err != nil {
		s.Status=500
		s.Result["error"]=err
		//panic(err)
		return
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err1 := file.Read(buffer)
	if err1 != nil && err1 != io.EOF {
		s.Status=500
		s.Result["error"]=err1
		//panic(err1)
		return
	}
	contentType = http.DetectContentType(buffer[:n])
	// end get content type

	img, err2 := imaging.Open(f)
	if err2 != nil {
		s.Status=500
		s.Result["error"]=err
		//panic(err)
		return
	}
	var thumb image.Image = imaging.Thumbnail(img, width, height, imaging.CatmullRom)

	buffer1 := new(bytes.Buffer)
	if contentType == "image/jpeg" {
		if err := jpeg.Encode(buffer1, thumb, nil); err != nil {
			s.Status=500
			s.Result["error"]="unable to encode image."
			//log.Println("unable to encode image.")
			return
		}
	}
	if contentType == "image/gif" {
		if err := gif.Encode(buffer1, thumb, nil); err != nil {
			s.Status=500
			s.Result["error"]="unable to encode image."
			//log.Println("unable to encode image.")
			return
		}
	}
	if contentType == "image/png"{
		if err := png.Encode(buffer1, thumb); err != nil {
			s.Status=500
			s.Result["error"]="unable to encode image."
			//log.Println("unable to encode image.")
			return
		}
	}

	w.Header().Set("Pragma", "cache")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Content-type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer1.Bytes())))

	if _, err := w.Write(buffer1.Bytes()); err != nil {
		s.Status=500
		s.Result["error"]="unable to write image."
		//log.Println("unable to write image.")
		return
	}
}

func  (s *Pump) imgresize(filepath, destinationFileName string)  {
	img, err := imaging.Open(filepath)
	if err != nil {
		/*panic(err)
		os.Exit(1)*/
		s.Status=500
		s.Result["error"]=err
		return
	}
	// resize image from  to 400 while preserving the aspect ration
	// Supported resize filters: NearestNeighbor, Box, Linear, Hermite, MitchellNetravali,
	// CatmullRom, BSpline, Gaussian, Lanczos, Hann, Hamming, Blackman, Bartlett, Welch, Cosine.
	dstimg := imaging.Resize(img, 100, 100, imaging.Box)
	// save resized image
	err = imaging.Save(dstimg, destinationFileName)
	if err != nil {
		/*panic(err)
		os.Exit(1)*/
		s.Status=500
		s.Result["error"]=err
		return
	}
	// everything ok
	return
}
func  (s *Pump) upload_base64(img string, w http.ResponseWriter, r *http.Request){
	//if r.Method!="POST"{return }

	//img:=r.FormValue("image")
	d1 := strings.SplitN(img, ",", 2)
	if !strings.Contains(d1[0], ";base64") {
		s.Status=406
		s.Result["error"]="invalid data"
		return
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(d1[1]))
	m, formatType, err := image.Decode(reader)
	if err != nil {
		s.Status=500
		s.Result["error"]=err
		return
	}
	//check file type
	if !(formatType == "jpg" || formatType == "jpeg" || formatType == "png" || formatType == "gif"){
		s.Status=500
		s.Result["error"]="invalid file type"
		return
	}

	//check image dimensions
	bounds := m.Bounds()
	if bounds.Dx() > 3000 || bounds.Dy() > 2000{
		return
	}

	var fileName = fmt.Sprintf("%d.%s", time.Now().UTC().UnixNano(), formatType)

	d2, err := base64.StdEncoding.DecodeString(d1[1])
	if err != nil {
		s.Status=500
		s.Result["error"]=err
		/*fmt.Println(err)
		panic(err)*/
		return
	}

	file:=s.open_file(fileName)
	defer file.Close()
	//***************

	_, err = file.Write(d2)
	if err != nil {
		s.Status=500
		s.Result["error"]=err
		/*fmt.Println(err)
		panic(err)*/
		return
	}

	/*if bounds.Dx() > 100 || bounds.Dy() > 100{
		s.imgresize(fileName,"thumb/"+fileName)
	}else {

	}*/
}

func  (s *Pump) upload(w http.ResponseWriter, r *http.Request){
	//if r.Method!="POST"{return }

	err := r.ParseMultipartForm(100000)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		s.Status=500
		s.Result["error"]=err
		return
	}
	//get a ref to the parsed multipart form
	m := r.MultipartForm

	//get the *fileheaders
	files := m.File["files[]"]
	for i, _ := range files {
		//for each fileheader, get a handle to the actual file
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			s.Status=500
			s.Result["error"]=err
			return
		}
		//create destination file making sure the path is writeable.
		var filename string = files[i].Filename

		dst:=s.open_file(filename)
		defer dst.Close()
		//copy the uploaded file to the destination file
		if _, err := io.Copy(dst, file); err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			s.Status=500
			s.Result["error"]=err
			return
		}
		//s.imgresize(filename,"thumb/"+filename)


	}
}


func  form(w http.ResponseWriter, r *http.Request){

	w.Write([]byte(`
<html><body>
	<h2>Multipart</h2>
	<form action="/" method="post" enctype="multipart/form-data">
    	<input type="file" name="files[]" multiple>
    	<input type="submit" value="Отправить">
	</form>


	<h2>upload_base64</h2>
	<form action="/" method="post" >
    	<input type="text" name="image" value="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==">
    	<input type="submit" value="Отправить">
	</form>

	<h2>upload_url</h2>
	<form action="/" method="post" >
    	<input type="text" name="url" value="http://i.imgur.com/m1UIjW1.jpg">
    	<input type="submit" value="Отправить">
	</form>

</body>
</html>

`))
}