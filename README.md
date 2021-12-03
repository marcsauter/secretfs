# secretfs

Package secretfs implements afero.Fs and afero.File for Kubernetes secrets.

Check: https://pkg.go.dev/golang.org/x/tools/godoc/vfs/mapf

A Kubernetes secret path can be written as */NAMESPACE/SECRET[/KEY]*. Where */NAMESPACE/SECRET* represents the directory and *KEY* the file part of the path.

