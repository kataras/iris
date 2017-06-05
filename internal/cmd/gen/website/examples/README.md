# Examples generate command

## Running methods

### Build and run each time
```sh
$ cd $GOPATH/src/github.com/kataras/iris/internal/cmd
$ go run main.go gen website examples >> recipe_content.html
``` 

### Using an executable
```sh
$ cd $GOPATH/src/github.com/kataras/iris/internal/cmd
$ go build
# rename the binary, executable file, to something like "iris" or "iris.exe" for win
# copy it to your systems folder or to the $GOPATH/bin
```
And use that command instead:
```sh
$ iris gen website examples >> recipe_content.html 
```
> That executable can be copied and used anywhere.

## Action

This command should write to the argument output `>>` or print to the `os.Stdout` something like this:
```html
    <h2 id="Beginner"><a href="#Beginner" class="headerlink" title="Beginner"></a>Beginner</h2>
    <h3 id="Hello-World"><a href="#Hello-World" class="headerlink" title="Hello World"></a>Hello World</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/hello-world/main.go" data-visible="true" class ="line-numbers codepre"></pre>

    <h3 id="Overview"><a href="#Overview" class="headerlink" title="Overview"></a>Overview</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/overview/main.go" data-visible="true" class ="line-numbers codepre"></pre>

    <h3 id="Internal-Application-File-Logger"><a href="#Internal-Application-File-Logger" class="headerlink" title="Internal Application File Logger"></a>Internal
        Application File Logger</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/file-logger/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Write-JSON"><a href="#Write-JSON" class="headerlink" title="Write JSON"></a>Write JSON</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/write-json/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Read-JSON"><a href="#Read-JSON" class="headerlink" title="Read JSON"></a>Read JSON</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/read-json/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Read-Form"><a href="#Read-Form" class="headerlink" title="Read Form"></a>Read Form</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/read-form/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Favicon"><a href="#Favicon" class="headerlink" title="Favicon"></a>Favicon</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/favicon/main.go" class ="line-numbers codepre"></pre>

    <h3 id="File-Server"><a href="#File-Server" class="headerlink" title="File Server"></a>File Server</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/file-server/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Send-Files"><a href="#Send-Files" class="headerlink" title="Send Files"></a>Send Files</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/send-files/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Stream-Writer"><a href="#Stream-Writer" class="headerlink" title="Stream Writer"></a>Stream Writer</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/stream-writer/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Listen-UNIX-Socket"><a href="#Listen-UNIX-Socket" class="headerlink" title="Listen UNIX Socket"></a>Listen UNIX Socket</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/listen-unix/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Listen-TLS"><a href="#Listen-TLS" class="headerlink" title="Listen TLS"></a>Listen TLS</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/listen-tls/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Listen-Letsencrypt-Automatic-Certifications"><a href="#Listen-Letsencrypt-Automatic-Certifications" class="headerlink" title="Listen Letsencrypt (Automatic Certifications)"></a>Listen
        Letsencrypt (Automatic Certifications)</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/beginner/listen-letsencrypt/main.go" class ="line-numbers codepre"></pre>

    <h2 id="Intermediate"><a href="#Intermediate" class="headerlink" title="Intermediate"></a>Intermediate</h2>
    <h3 id="Send-an-email"><a href="#Send-an-email" class="headerlink" title="Send an email"></a>Send an email</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/e-mail/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Upload-Read-Files"><a href="#Upload-Read-Files" class="headerlink" title="Upload/Read Files"></a>Upload/Read Files</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/upload-files/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Request-Logger"><a href="#Request-Logger" class="headerlink" title="Request Logger"></a>Request Logger</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/request-logger/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Profiling-pprof"><a href="#Profiling-pprof" class="headerlink" title="Profiling (pprof)"></a>Profiling (pprof)</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/pprof/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Basic-Authentication"><a href="#Basic-Authentication" class="headerlink" title="Basic Authentication"></a>Basic Authentication</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/basicauth/main.go" class ="line-numbers codepre"></pre>

    <h3 id="HTTP-Access-Control"><a href="#HTTP-Access-Control" class="headerlink" title="HTTP Access Control"></a>HTTP Access Control</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/cors/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Cache-Markdown"><a href="#Cache-Markdown" class="headerlink" title="Cache Markdown"></a>Cache Markdown</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/cache-markdown/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Localization-and-Internationalization"><a href="#Localization-and-Internationalization" class="headerlink" title="Localization and Internationalization"></a>Localization
        and Internationalization</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/i18n/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Recovery"><a href="#Recovery" class="headerlink" title="Recovery"></a>Recovery</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/recover/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Graceful-Shutdown"><a href="#Graceful-Shutdown" class="headerlink" title="Graceful Shutdown"></a>Graceful Shutdown</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/graceful-shutdown/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Custom-TCP-Listener"><a href="#Custom-TCP-Listener" class="headerlink" title="Custom TCP Listener"></a>Custom TCP Listener</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/custom-listener/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Custom-HTTP-Server"><a href="#Custom-HTTP-Server" class="headerlink" title="Custom HTTP Server"></a>Custom HTTP Server</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/custom-httpserver/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Password-Hashing"><a href="#Password-Hashing" class="headerlink" title="Password Hashing"></a>Password Hashing</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/password-hashing/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Flash-Messages"><a href="#Flash-Messages" class="headerlink" title="Flash Messages"></a>Flash Messages</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/intermediate/flash-messages/main.go" class ="line-numbers codepre"></pre>

    <h2 id="Advance"><a href="#Advance" class="headerlink" title="Advance"></a>Advance</h2>
    <h3 id="Transactions"><a href="#Transactions" class="headerlink" title="Transactions"></a>Transactions</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/transactions/main.go" class ="line-numbers codepre"></pre>

    <h3 id="HTTP-Testing"><a href="#HTTP-Testing" class="headerlink" title="HTTP Testing"></a>HTTP Testing</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/httptest/main_test.go" class ="line-numbers codepre"></pre>

    <h3 id="Watch-amp-Compile-Typescript-source-files"><a href="#Watch-amp-Compile-Typescript-source-files" class="headerlink" title="Watch &amp; Compile Typescript source files"></a>Watch
        &amp; Compile Typescript source files</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/typescript/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Cloud-Editor"><a href="#Cloud-Editor" class="headerlink" title="Cloud Editor"></a>Cloud Editor</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/cloud-editor/main.go" class ="line-numbers codepre"></pre>

    <h3 id="Online-Visitors"><a href="#Online-Visitors" class="headerlink" title="Online Visitors"></a>Online Visitors</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/online-visitors/main.go" class ="line-numbers codepre"></pre>

    <h3 id="URL-Shortener-using-BoltDB"><a href="#URL-Shortener-using-BoltDB" class="headerlink" title="URL Shortener using BoltDB"></a>URL Shortener using BoltDB</h3>
    <pre data-src="https://raw.githubusercontent.com/kataras/iris/master/_examples/advanced/url-shortener/main.go" class ="line-numbers codepre"></pre>
```