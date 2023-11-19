# Prisma Go Demo

## Instructions

```shell script
git clone git@github.com:steebchen/prisma-go-demo.git
cd prisma-go-demo
go run github.com/steebchen/prisma-client-go db push
go run .
# created post: {
#   "id": "ckfnrp7ec0000oh9kygil9s94",
#   "createdAt": "2020-09-29T09:37:44.628Z",
#   "updatedAt": "2020-09-29T09:37:44.628Z",
#   "title": "Hi from Prisma!",
#   "published": true,
#   "desc": "Prisma is a database toolkit and makes databases easy."
# }
# post: {
#   "id": "ckfnrp7ec0000oh9kygil9s94",
#   "createdAt": "2020-09-29T09:37:44.628Z",
#   "updatedAt": "2020-09-29T09:37:44.628Z",
#   "title": "Hi from Prisma!",
#   "published": true,
#   "desc": "Prisma is a database toolkit and makes databases easy."
# }
# The posts's title is: Prisma is a database toolkit and makes databases easy.
```

## Next steps

Read the docs at [GoPrisma](https://goprisma.org/docs).
