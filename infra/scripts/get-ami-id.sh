#! /bin/bash

aws ec2 describe-images \
--owners "aws-marketplace" \
--filters \
"Name=name,Values=*openSUSE*" \
"Name=architecture,Values=arm64" \
"Name=state,Values=available" \
--query "sort_by(Images, &CreationDate)[-5:].{ID:ImageId,Name:Name,Date:CreationDate}" \
--output table \
--region us-east-1