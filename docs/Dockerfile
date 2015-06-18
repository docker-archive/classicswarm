FROM docs/base:hugo
MAINTAINER Mary Anthony <mary@docker.com> (@moxiegirl)

# to get the git info for this repo
COPY . /src

COPY . /docs/content/swarm/

# Sed to process GitHub Markdown
# 1-2 Remove comment code from metadata block
# 3 Change ](/word to ](/project/ in links
# 4 Change ](word.md) to ](/project/word)
# 5 Remove .md extension from link text
# 6 Change ](./ to ](/project/word) 
# 7 Change ](../../ to ](/project/ 
# 8 Change ](../ to ](/project/ 
# 
RUN find /docs/content/swarm -type f -name "*.md" -exec sed -i.old \
    -e '/^<!.*metadata]>/g' \
    -e '/^<!.*end-metadata.*>/g' \
    -e 's/\(\]\)\([(]\)\(\/\)/\1\2\/swarm\//g' \
    -e 's/\(\][(]\)\([A-z].*\)\(\.md\)/\1\/swarm\/\2/g' \
    -e 's/\([(]\)\(.*\)\(\.md\)/\1\2/g'  \
    -e 's/\(\][(]\)\(\.\/\)/\1\/swarm\//g' \
    -e 's/\(\][(]\)\(\.\.\/\.\.\/\)/\1\/swarm\//g' \
    -e 's/\(\][(]\)\(\.\.\/\)/\1\/swarm\//g' {} \;

