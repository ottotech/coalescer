coalescer
=========

## Overview
coalescer is a CLI tool written in Go that allows you to recognize people in images from a directory and 
coalesce the pictures to a new location when there is a match.

coalescer uses under the hood [facebox](https://machinebox.io/docs/facebox) by [machinebox](https://machinebox.io/). 
This is an awesome tool for face recognition written in GO with a clear and easy-to-use api. coalescer uses facebox to 
leverage its core functionality.  

So in order to use coalescer you need to have a facebox instance running (more about this in the how-to-use part).

## How to use?

Before start, you have to install in your machine:
- +go 1.14
- Docker

##### **Step 1**
Sign-up in [machinebox](https://machinebox.io/) to get a free key, so you can use facebox (one of machinebox products) 
for free as a developer. 

##### **Step 2**
Once you have done all the setup with machinebox, and you have gotten your MB_KEY, 
you need to run a facebox instance. The easiest way to do that is to pull a docker image and use your key:
```
docker run -p 8080:8080 -e "MB_KEY=$MB_KEY" machinebox/facebox
```
where $MB_KEY should be the key you got from machinebox. 

##### **Step 3**
You need to install the coalescer code and command line program:
```
go get -u github.com/ottotech/coalescer...
```
##### **Step 4**
There you are. Now you are ready to use coalescer. Run ```coalescer -h``` if you want to know more about coalescer flag options: 
```
coalescer -h 
```
For more see the Usage part.

## Usage
So imagine you have a directory called "fotos" with the following sub-directories with images.
```
.
├── people_dir
│   ├── irene_1.jpeg
│   ├── irene_2.jpeg
│   ├── irene_3.jpeg
│   ├── irene_4.jpeg
│   ├── otto_1.png
│   └── otto_2.jpeg
└── pics_dir
    ├── irene_photo_x.jpg
    ├── irene_photo_y.jpg
    ├── irene_and_otto_x.jpg
    ├── otto_photo_x.jpg
    └── julia_photo.jpg
```
Here the *people_dir* will contain images of the people you want to recognize in the pictures inside *pics_dir*.
The images in *people_dir* should be images containing only the person you want to recognize, so there should
be only one face per image. Is ideal to have more than two images of the same person you want to recognize 
so that you can teach facebox more about each person (this is all for facebox to be able to recognize people accurately).
The naming of the files inside *people_dir* is also important. The names of the files should contain the name of the 
person we want to recognize, for example, the *irene* part, then an underscore followed by any identifier you want,
for example, the *1* part. This is very important because coalescer will use the names of the people from the filenames 
to uniquely identify each person in each picture inside *pics_dir*.
    
## **Example 1**

If you run coalescer with the following flags:
```
$ coalescer \
  -peopledir=people_dir \
  -picsdir=pics_dir \
  -faceboxurl=http://localhost:8080/ \
  -confidence=70 \
```
You will see that coalescer should have created two directories: irene and otto. 

Inside the irene directory you should be able to see the following pictures:
-  irene_photo_x.jpg
-  irene_photo_y.jpg
-  irene_and_otto_x.jpg

Inside the otto directory you should be able to see the following pictures:
-  otto_photo_x.jpg
-  irene_and_otto_x.jpg

So in this example, when you don't use the *-combine* flag coalescer will grab all the people you want to recognize from
the *people_dir* and will create a directory for each of them where it's gonna copy all the images that match their faces.

## **Example 2**

If you run coalescer with the following flags:
```
$ coalescer \
  -peopledir=people_dir \
  -picsdir=pics_dir \
  -faceboxurl=http://localhost:8080/ \
  -confidence=70 \
  -combine=irene,otto
```
You will see that coalescer should have created only one directory with the name of the two people you want to 
recognize in each picture. The directory name in this case will be: irene_otto. 

Inside the irene_otto directory you should see the following picture:
-  irene_and_otto_x.jpg

So in this example, when you use the *-combine* flag coalescer will check if each picture inside *pics_dir* has the faces
of all the people defined inside *people_dir*. coalescer will use an AND gate logic to filter out the pictures. If there
is a match coalescer will copy each picture to a single folder which has a name composed by all the names of the people
you want to recognize.

---

So I hope with this you get an idea of what coalescer can do.  

## Contributing

If you make any changes, run ```go fmt ./...``` before submitting a pull request.

## Notes

- coalescer won't remove your pictures from *pics_dir* (e.g. if you followed the Usage part). It will only copy the images
to a new location. However, if you are dealing with important pictures I'll advise you to have always a backup before 
using coalescer.

- Remember to always have running your facebox instance before using coalescer, since coalescer depends on it.

- facebox is not fully free, for developers and open source projects, it has a limit of 100 faces to recognize. For the use
cases of coalescer that's more than enough. If you of course need a higher limit you can consider upgrade your machinebox
account.

- coalescer uses some concurrent and parallelism patterns described in [this go blog](https://blog.golang.org/pipelines)

- I won't be actively improving this repo, but from time to time I will try to enhance it :)

## TODO
- Tests (at this moment there are no tests, will try to find time to add them soon).
- The logic inside the recognizeAndCopy func can be simplified.  

## License

Copyright ©‎ 2020, [ottotech](https://ottotech.site/)

Released under MIT license, see [LICENSE](https://github.com/ottotech/coalescer/blob/master/LICENSE.md) for details.

