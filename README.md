If using Docker

create directory

`$ mkdir Pump`

Deploy project files to the folder

`$ cd Pump`

`$ docker build -t pump .`

`$ docker run --publish 6060:3008 --name test --rm pump
`

http://localhost:6060/form

