#!/bin/bash

# Returns the version tag from current commit
# If current commit has no version tag but is a merge then
# check its parents and pick the higher
# If all the above fails then use commit hash

HEAD_TAGS=$(git describe --tags HEAD)

if [[ "$HEAD_TAGS" =~ ^v[0-9]+.[0-9]+.[0-9]+$ ]]
then
    echo "$HEAD_TAGS"
else
    LEFT_TAGS=$(git describe --tags HEAD^1)
    RIGHT_TAGS=$(git describe --tags HEAD^2 2> /dev/null)
    if [ $? != 0 ]
    then
        echo $(git log -1 --pretty=%h)
        exit 0
    fi
    if [[ "$LEFT_TAGS" =~ ^v[0-9]+.[0-9]+.[0-9]+$ ]]
    then
        if [[ "$RIGHT_TAGS" =~ ^v[0-9]+.[0-9]+.[0-9]+$ ]]
        then
            if [ "$LEFT_TAGS" \> "$RIGHT_TAGS" ]
            then
                echo "$LEFT_TAGS"
            else
                echo "$RIGHT_TAGS"
            fi
        else
            echo "$LEFT_TAGS"
        fi
    else
        if [[ "$RIGHT_TAGS" =~ ^v[0-9]+.[0-9]+.[0-9]+$ ]]
        then
            echo "$RIGHT_TAGS"
        else
            echo $(git log -1 --pretty=%h)
        fi
    fi
fi