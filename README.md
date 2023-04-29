# Imagen a vector
Automatiza la transformación de imágenes en jpg y png a SVG a través del servicio de vectorización que ofrece la web de [Adobe Express](https://www.adobe.com/es/express/feature/image/convert/svg).


## Funcionamiento
Descargar de la carpeta bin el binario correspondiente al sistema operativo que se vaya a utilizar. Asegurar que el archivo dispone de permisos de ejecución.\
Argumentos:\
- **-ruta**: obligatorio, ruta completa al archivo txt con las rutas completas línea a línea de cada imagen que quiere procesarse.
- **-espera**: opcional, segundos de espera antes de una nueva petición de transformación.
- **-apikey**: opcional, valor que se incorpora en las cabeceras de la petición con la clave "X-Api-Key" y valida la petición
```bash
./imagen_a_vector -ruta /home/user/go/src/imagen_a_vector/imagenes.txt
```

Contenido del archivo imagenes.txt
```
/home/user/go/src/imagen_a_vector/bin/go_gopher.jpg
/home/user/go/src/imagen_a_vector/bin/go_logo.png
```
