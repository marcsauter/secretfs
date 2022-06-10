# secfs

Package secfs implements afero.Fs and afero.File for Kubernetes secrets.

A Kubernetes secret path can be written as */NAMESPACE/SECRET[/KEY]*. Where */NAMESPACE/SECRET* represents the directory and *KEY* the file part of the path.

