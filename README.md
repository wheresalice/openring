# openring

This is a tool for generating a webring from RSS feeds, so you can link to
other blogs you like on your own blog. It's designed to be fairly simple and
integrate with any static site generator. The basic usage is:

```
openring sites.json < in.html > out.html
```

This will read the template at in.html (an example is provided, but feel free to
adjust it to suit your needs), fetch the latest 3 articles from among your
sources, and pass them to the template and write the output to out.html. Then
you can include this file with your static site generator's normal file include
mechanism.
