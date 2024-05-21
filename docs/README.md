Gwyneth
===

**Gwyneth** is an RSS proxy that facilitates the retrieval and filtering of RSS feeds.  
It allows for the execution of scripts associated with filters.  

* **RSS Retrieval**: gwyneth can fetch RSS feeds from various sources.
* **Filtering**: It enables filtering of RSS feeds based on specified criteria.
* **Script Execution**: Associated scripts can be triggered to perform actions based on filter results.

![overview](./imgs/overview.png)  

Gwyneth's main function is that it is a feed proxy. There is a collector that collects the feeds and a DB that stores the originals.  
When storing the feeds, they can be passed through a pre-registered filter, and if they match the filter, an arbitrary script registered as an action can be fired.  
Scripts are directly called from those installed on the OS, so there are no restrictions on their operation.  
For example, it can be used to notify Slack by hitting a Webhook.  

Also, by executing Gwyneth's own API, new and unique feeds can be generated.  
Gwyneth has original feeds and mixed feeds as its domain, and by registering articles with specific conditions in the mixed feeds, it is possible to generate arbitrary feeds.  

## USAGE

API details are as per Swagger.  
Gwyneth's samples and Swagger can be activated as follows.  

```bash
cd docker
make
```

* gwyneth
	* http://localhost:8000/gwyneth/
* swagger
	* http://localhost:8000/swagger/


## Settings [sample](../docker/etc/gwyneth.yaml)
```
database:
  host: <database's host>
  port: <port>
  database: <database's name>
  user: <user>
  password: <password>
http:
  host: <api's listen address>
  port: <api's listen port>
feed: <Setting up Feeds to be delivered>
  title: <feed's title>
  description: <feed's description>
  link: <link>
  author_name: <name>
  author_email: <email>
  default_type: <default feed type. (rss / json / atom)>
```
