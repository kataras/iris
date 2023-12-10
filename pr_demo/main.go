package main

import "fmt"

/*
# 将fork出来的仓库clone到本地（自己的仓库）
git clone git@github.com:xxxxx/local.git
# 查看仓库情况
git remote -v
# 添加自己fork的那个仓库（公司的仓库地址）
git remote add upstream git@github.xxxxx/company.git

# 从upstream/main分支上新建一个upstream-local本地分支，并切换到该分支
git checkout -b upstream-local upstream/main

# 将upsream/main分支上的提交rebase(同步)到本地
git fetch upstream
git rebase upstream/main

	# 将本地提交push到远程，然后我们就可以在github的upstream-loca分支上提交pr了
	git push --set-upstream origin

	# 拉取origin对应分支的代码
	#  git pull origin 1.4.000
*/
func main() {
	fmt.Println("pr demo...")
}
