package main

import (
    "fmt"
    "github.com/gin-contrib/static"
    "github.com/gin-gonic/gin"
    "github.com/skip2/go-qrcode"
    "io"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
    "os/exec"
    "path"
    "runtime"
    "time"
)

var filesDirName = "upload_pics"
var filePrefix = "ul"

func main() {
    r := gin.Default()
    r.Use(static.Serve("/assets/", static.LocalFile("assets", false)))
    serverAddr := fmt.Sprintf("%s:%d", getOutboundIP().String(), 8080)

    pwd, _ := os.Getwd()
    filesDirName = path.Join(pwd, filesDirName)

    r.GET("/", func(c *gin.Context) {
        file, err := os.Open("index.html")
        if err != nil {
            log.Fatal(err)
        }
        homePage, _ := ioutil.ReadAll(file)
        file.Close()
        c.Writer.Write(homePage)
    })
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "ok",
        })
    })
    r.GET("/urlqrcode", func(c *gin.Context) {
        png, err := qrcode.Encode("http://"+serverAddr, qrcode.Medium, 256)
        if err != nil {
            log.Printf("%v\n", err)
            c.Status(http.StatusInternalServerError)
            return
        }
        c.Writer.Write(png)
    })
    r.POST("/upload", handleUploadMutiFile)

    setopenbrowser(serverAddr)

    if err := r.Run(serverAddr); err != nil {
        log.Fatal(err)
    }
}

func setopenbrowser(serverAddr string) {
    go func() {
        reqUrl := fmt.Sprintf("http://%s/health", serverAddr)
        for {
            var (
                client = &http.Client{Timeout: time.Second * 5}
                err    error
                resp   *http.Response
            )
            if resp, err = client.Get(reqUrl); err != nil {
                continue
            }
            defer resp.Body.Close()
            openbrowser("http://" + serverAddr)
            break
        }
    }()
}

func getOutboundIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP
}

func openbrowser(url string) {
    var err error

    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    default:
        err = fmt.Errorf("unsupported platform")
    }
    if err != nil {
        log.Fatal(err)
    }
}

func handleUploadMutiFile(c *gin.Context) {
    if _, err := os.Stat(filesDirName); err != nil {
        if os.IsNotExist(err) {
            _ = os.Mkdir(filesDirName, 0777)
        }
    }
    // 设置文件大小
    err := c.Request.ParseMultipartForm(4 << 20)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"msg": "文件太大"})
        return
    }
    formdata := c.Request.MultipartForm
    files := formdata.File["uploadfiles"]

    for _, v := range files {
        file, err := v.Open()
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"msg": "文件读取失败"})
            return
        }
        defer file.Close()

        filename := fmt.Sprintf("%s_%d_%s", filePrefix, time.Now().Unix(), v.Filename)
        storePath := path.Join(filesDirName, filename)
        w, err := os.Create(storePath)
        if err != nil {
           c.JSON(http.StatusInternalServerError, gin.H{"msg": "创建文件失败", "file": storePath})
           return
        }
        defer w.Close()
        if _, err := io.Copy(w, file); err != nil {
           c.JSON(http.StatusInternalServerError, gin.H{"msg": "写入文件失败", "file": v.Filename})
           return
        }
    }

    c.JSON(http.StatusOK, gin.H{"msg": "上传成功"})
}
