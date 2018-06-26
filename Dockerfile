FROM golang:1.9.4

WORKDIR /go/src/logstatd
COPY . .

RUN mkdir dist &&\
	mkdir dist/conf

RUN go env

RUN go build -o dist/logstatd &&\
    cp -f conf/logstatd.ini dist/conf/logstatd.ini &&\
	tar -zcvf dist.tgz dist/* 

#	scp dist.tgz halo_op@106.75.24.243:/data/golang-data

RUN ls -l && ls -l dist

RUN cp -f dist/logstatd /usr/local/bin/ &&\
	cp -f dist/conf/logstatd.ini /usr/local/etc/

CMD ["/usr/local/bin/logstatd", "-c", "/usr/local/etc/logstatd.ini" ] 
