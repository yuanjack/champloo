champloo  [![wercker status](https://app.wercker.com/status/fc23dcf2cd76e92e4f20a1e761793871/s "wercker status")](https://app.wercker.com/project/bykey/fc23dcf2cd76e92e4f20a1e761793871)
========

A simple project for deploy php website.

![Preview](http://ww2.sinaimg.cn/large/7ce4a9f6gw1ekc1bg5ljyj20o40ggwfb.jpg)

## Features

- Support svn & git
- Easy to start

## Installation

- Get lastest version in [Release](https://github.com/cxfksword/champloo/releases) page. 
- Install and start the web:
```
$ ./champloo
```
- Login into website ```http://localhost:3000``` with default account:```admin```, password:```123```

## Install client

- In the deploy server execute shell script:
```
$ wget https://raw.githubusercontent.com/cxfksword/champloo-client/master/install.sh
$ chmod +x install.sh
$ ./install.sh
```
- Start client service
```
$ cd /usr/local/champloo-client
$ cp app.conf.example app.conf       # change the api point to corrent address
$ service champloo-client start
```

## Todo

- Web console panel
- Show comments after last rev.
- Email notification & webhook
