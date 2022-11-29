for f in $(find ./ -name '*.go'); do
  goimports -w $f
done
