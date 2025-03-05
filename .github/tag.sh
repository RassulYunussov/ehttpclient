#!/bin/bash
next_version=v0.0.$(git tag --sort=-v:refname | head -n 1 | sed 's|.*\.||' | awk '{print $1 + 1}')
echo $next_version
#echo $next_version
#git tag $next_version
#gh release create $next_version
#go list -m github.com/RassulYunussov/ehttpclient@$next_version