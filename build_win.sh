if [ ! -d "build" ]
then
  mkdir build
fi

GOOS=windows GOARCH=amd64 go build -o build/upload-photo.exe .
cp index.html build/
cp -r assets build/assets
if [ ! -d "build/upload_pics" ]
then
  mkdir build/upload_pics
fi
