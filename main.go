package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func extraerSVGCode(contenido string) (string, error) {
	expRegSVG := regexp.MustCompile(`(?m)(?s).*?(<svg.*</svg>).*`)
	if !expRegSVG.MatchString(contenido) {
		return "", errors.New("no se ha encontrado el código svg en la respuesta")
	}
	return expRegSVG.ReplaceAllString(contenido, "$1"), nil
}

func añadirCabecerasGenericas(peticion *http.Request) {
	cabeceras := map[string]string{
		"User-Agent":      "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0",
		"Accept":          "*/*",
		"Accept-Language": "es-ES,en;q=0.5",
		"Referer":         "https://express.adobe.com/",
		"Prefer":          "respond-async, wait=180",
		"Origin":          "https://express.adobe.com",
		"Connection":      "keep-alive",
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "cross-site",
	}

	for c, v := range cabeceras {
		peticion.Header.Set(c, v)
	}
}

func extensionPermitida(extension string) bool {
	extensionesValidas := []string{"png", "jpg", "jpeg"}
	for _, v := range extensionesValidas {
		if v == extension {
			return true
		}
	}
	return false
}

func main() {

	var ruta string
	flag.StringVar(&ruta, "ruta", "", "ruta completa al archivo de texto con la relación de imágenes a vectorizar")

	var espera int
	flag.IntVar(&espera, "espera", 0, "segundos que se esperarán entre cada solicitud de conversión")

	var apiKey string
	flag.StringVar(&apiKey, "apikey", "MarvelWeb3", "apik key que se incorpora en la cabecera de la petición")

	flag.Parse()

	if ruta == "" {
		log.Fatalln("No se ha proporcionado la ruta del archivo")
	}

	archivo, archivoError := os.Open(ruta)
	if archivoError != nil {
		log.Fatalf("No se ha podido abrir el archivo proporcionado: %v\n", archivoError.Error())
	}

	cliente := &http.Client{
		Timeout: time.Second * 10,
	}

	escaner := bufio.NewScanner(archivo)
	escaner.Split(bufio.ScanLines)
	var indice int

	var errores []string
	for escaner.Scan() {
		if indice > 0 {
			time.Sleep(time.Duration(espera) * time.Second)
		}
		indice++
		imagenRuta := strings.TrimSpace(escaner.Text())
		if imagenRuta == "" {
			continue
		}
		fmt.Printf("===> Procesando %v, por favor, espere...\n", imagenRuta)
		imagenRutaBase := filepath.Dir(imagenRuta)

		imagenExtension := strings.TrimPrefix(filepath.Ext(imagenRuta), ".")
		if !extensionPermitida(imagenExtension) {
			mensajeError := fmt.Sprintf("%v (línea %d) extensión de imagen no permitida (solo png y jpg)\n", imagenRuta, indice)
			errores = append(errores, mensajeError)
			continue
		}
		imagenNombreExtension := filepath.Base(imagenRuta)
		imagenNombre := strings.TrimSuffix(imagenNombreExtension, filepath.Ext(imagenRuta))

		vectorNombre := fmt.Sprintf("%v.svg", imagenNombre)
		vectorRuta := filepath.Join(imagenRutaBase, vectorNombre)

		/* PREPARAR PETICIÓN */
		cuerpo := &bytes.Buffer{}
		manejadorMultipart := multipart.NewWriter(cuerpo)
		campoForm, campoFormError := manejadorMultipart.CreateFormField("contentAnalyzerRequests")
		if campoFormError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al crear el campo contentAnalyzerRequests: %v\n", imagenRuta, indice, campoFormError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		if imagenExtension == "jpg" {
			imagenExtension = "jpeg"
		}

		campoInfo := fmt.Sprintf(`{"sensei:name":"Vectorize Service","sensei:invocation_mode":"asynchronous","sensei:invocation_batch":false,"sensei:in_response":false,"sensei:engines":[{"sensei:execution_info":{"sensei:engine":"Feature:vectorize-service:358228e1-e9d9-4600-beb4-70e38af4c600"},"sensei:inputs":{"image_in":{"dc:format":"image/%v","sensei:multipart_field_name":"inFile"}},"sensei:outputs":{"svg_out":{"dc:format":"image/svg+xml","sensei:multipart_field_name":"outFileSvg"}}}]}`, imagenExtension)

		_, CampoInfoError := campoForm.Write([]byte(campoInfo))
		if CampoInfoError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al escribir la información del campo: %v\n", imagenRuta, indice, CampoInfoError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		campoFichero, campoFicheroError := manejadorMultipart.CreateFormFile("inFile", imagenNombre)
		if campoFicheroError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al escribir el campo fichero: %v\n", imagenRuta, indice, campoFicheroError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		archivoImagen, archivoImagenError := os.Open(imagenRuta)
		if archivoImagenError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al leer la imagen: %v\n", imagenRuta, indice, archivoImagenError.Error())
			errores = append(errores, mensajeError)
			continue
		}
		_, copiaError := io.Copy(campoFichero, archivoImagen)
		if copiaError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error copiando la imagen: %v\n", imagenRuta, indice, archivoImagenError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		cierreError := manejadorMultipart.Close()
		if cierreError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error cerrando el manejador multipart: %v\n", imagenRuta, indice, cierreError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		peticion, peticionError := http.NewRequest("POST", "https://sensei.adobe.io/anonymous/v2/predict/", bytes.NewReader(cuerpo.Bytes()))
		if peticionError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error preparación la petición: %v\n", imagenRuta, indice, peticionError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		añadirCabecerasGenericas(peticion)
		peticion.Header.Set("Content-Type", manejadorMultipart.FormDataContentType())
		peticion.Header.Set("X-Api-Key", apiKey)

		respuesta, respuestaError := cliente.Do(peticion)
		if respuestaError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al realizar la petición: %v\n", imagenRuta, indice, respuestaError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		if respuesta.StatusCode != 200 {
			mensajeError := fmt.Sprintf("%v (línea %d) error al recibir un status code incorrecto: %v\n", imagenRuta, indice, respuesta.Status)
			errores = append(errores, mensajeError)
			continue
		}

		respuestaContenido, respuestaContenidoError := io.ReadAll(respuesta.Body)
		if respuestaContenidoError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al leer la respuesta: %v\n", imagenRuta, indice, respuestaContenidoError.Error())
			errores = append(errores, mensajeError)
			continue
		}

		contenido := string(respuestaContenido)
		codigoSVG, codigoSVGError := extraerSVGCode(contenido)
		if codigoSVGError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al no encontrar código svg en la respuesta\n", imagenRuta, indice)
			errores = append(errores, mensajeError)
			continue
		}

		archivoSVG, archivoSVGError := os.Create(vectorRuta)
		if archivoSVGError != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al crear el archivo .svg: %v\n", imagenRuta, indice, archivoImagenError.Error())
			errores = append(errores, mensajeError)
			continue
		}
		defer archivoSVG.Close()

		_, errorEscritura := archivoSVG.Write([]byte(codigoSVG))
		if errorEscritura != nil {
			mensajeError := fmt.Sprintf("%v (línea %d) error al escribir el archivo .svg: %v\n", imagenRuta, indice, errorEscritura.Error())
			errores = append(errores, mensajeError)
			continue
		}
	}

	if len(errores) > 0 {
		log.Println("¡¡ATENCIÓN!! Ejecución con errores:")
		for _, v := range errores {
			log.Print(v)
		}
	}
}
