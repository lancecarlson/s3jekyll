s3jekyll
========

Amazon S3 uploader for jekyll sites

download
========

More platforms coming soon!

OS X:
http://labs.healpay.com/s3jekyll/bin/s3jekyll_darwin

move the binary file into one of your bin directories (/usr/bin/local works)

usage
=====

run the s3jekyll command inside of your jekyll site for the first time and it will create an example config file for you. It's called .production.s3.json. It should look something like this:

```javascript
{
    "access": "",
    "secret": "",
    "bucket": "",
    "from": "_site"
}
```

fill in the correct config values and run the s3jekyll command again and it will upload your _site folder into the root of the bucket.

configuration
=============

Coming soon!
