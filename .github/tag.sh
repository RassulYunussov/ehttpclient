#!/bin/bash
initial_version=$(git tag --sort=-creatordate | head -n 1)
prefix_version=$(echo $initial_version | sed 's/[^.]*$//')
next_version=$prefix_version$(echo $initial_version | sed 's|.*\.||' | awk '{print $1 + 1}')
git tag $next_version
git push origin tag $next_version
gh release create $next_version
go list -m github.com/RassulYunussov/ehttpclient@$next_version