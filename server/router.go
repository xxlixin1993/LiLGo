package server

import (
	"github.com/xxlixin1993/LiLGo/utils"
)

type (
	Router struct {
		tree *node
	}

	node struct {
		// 节点路径，比如search和see 提取se
		prefix string
		// :变量 append(匹配的值) , * 全部匹配时append(*)
		matchParam []string
		// url请求原始路径
		realPath string
		// 节点类型，staticNodeType, paramNodeType, anyNodeType
		// static: 静态节点，比如上面的s，earch等节点
		// any: 有*匹配的节点
		// param: 参数节点
		nodeType nodeType
		// 儿子节点
		children []*node
		// 节点第一个字符
		// 和children字段对应, 保存的是分裂的分支的第一个字符
		// 例如search和support, 那么s节点的indices对应的"eu"
		// 代表有两个分支, 分支的首字母分别是e和u
		firstChar byte
		// 父亲节点
		parent *node
		// 处理函数 这个path支持的所有http method
		handlers *methodHandler
	}

	methodHandler struct {
		connect HandlerFunc
		delete  HandlerFunc
		get     HandlerFunc
		head    HandlerFunc
		options HandlerFunc
		patch   HandlerFunc
		post    HandlerFunc
		put     HandlerFunc
		trace   HandlerFunc
	}

	// static: 静态节点，比如上面的s，earch等节点
	// any: 有*匹配的节点
	// param: 参数节点
	nodeType uint8
)

const (
	staticNodeType nodeType = iota // default
	paramNodeType
	anyNodeType
)

func NewRouter() *Router {
	return &Router{
		tree: &node{
			handlers: new(methodHandler),
		},
	}
}

func (r *Router) Add(method, path string, h HandlerFunc) {
	// Validate registered path
	if path == "" {
		panic("echo: path cannot be empty")
	}

	// Unified path format
	if path[0] != '/' {
		path = "/" + path
	}

	var matchParam []string

	for i, length := 0, len(path); i < length; i++ {
		if path[i] == ':' {
			// match :
			r.insert(method, path[:i], nil, staticNodeType, "", nil)

			j := i + 1
			for ; i < length && path[i] != '/'; i++ {
			}

			matchParam = append(matchParam, path[j:i])
			path = path[:j] + path[i:]
			i, length = j, len(path)

			if i == length {
				r.insert(method, path[:i], h, paramNodeType, path, matchParam)
				return
			}

			r.insert(method, path[:i], nil, paramNodeType, path, matchParam)
		} else if path[i] == '*' {
			// match *
			r.insert(method, path[:i], nil, staticNodeType, "", nil)

			matchParam = append(matchParam, "*")
			r.insert(method, path[:i+1], h, anyNodeType, path, matchParam)
			return
		}
	}

	r.insert(method, path, h, staticNodeType, path, matchParam)
}

func (r *Router) insert(method, insertPath string, h HandlerFunc, nodeT nodeType, realPath string, matchParam []string) {
	if r.tree == nil {
		panic("plz init router first")
	}

	// 当前node指针 初始化是为root
	currentNode := r.tree
	for {
		insertPathLen := len(insertPath)
		prefixLen := len(currentNode.prefix)
		maxCycles := utils.MinInt(insertPathLen, prefixLen)
		searchIndex := 0

		for ; searchIndex < maxCycles && insertPath[searchIndex] == currentNode.prefix[searchIndex]; searchIndex++ {
		}

		if searchIndex == 0 {
			// insertPath is root node
			currentNode.firstChar = insertPath[0]
			currentNode.prefix = insertPath
			if h != nil {
				currentNode.nodeType = nodeT
				currentNode.addHandler(method, h)
				currentNode.realPath = realPath
				currentNode.matchParam = matchParam
			}
		} else if searchIndex < prefixLen {
			// insertPath 和 当前节点匹配的prefix可以再细分
			// ex: insertPath 是 seeMovie， 而tree现在节点上的相同点是seeA 所以可以把tree上的相同点变成see
			n := newNode(currentNode.nodeType, currentNode.prefix[searchIndex:], currentNode, currentNode.children,
				currentNode.handlers, currentNode.realPath, currentNode.matchParam)

			// Reset parent node
			currentNode.nodeType = staticNodeType
			currentNode.firstChar = currentNode.prefix[0]
			currentNode.prefix = currentNode.prefix[:searchIndex]
			currentNode.children = nil
			currentNode.handlers = new(methodHandler)
			currentNode.realPath = ""
			currentNode.matchParam = nil

			currentNode.addChild(n)

			if searchIndex == insertPathLen {
				// 此次插入是一个父节点 即找不到可以挂载的父节点 无法成为子节点
				currentNode.nodeType = nodeT
				currentNode.addHandler(method, h)
				currentNode.realPath = realPath
				currentNode.matchParam = matchParam
			} else {
				// 此次插入是一个子节点 即找到了可以挂载的父节点
				n = newNode(nodeT, insertPath[searchIndex:], currentNode, nil, new(methodHandler), realPath, matchParam)
				n.addHandler(method, h)
				currentNode.addChild(n)
			}
		} else if searchIndex < insertPathLen {
			// 没匹配完所有的insertPath
			// ex: insertPath 是 /seeWorld ,匹配了/之后 知道应该挂载在/上 然后seeWorld重新找该插入的节点
			insertPath = insertPath[searchIndex:]

			c := currentNode.findChildWithFirstChar(insertPath[0])
			if c != nil {
				// Go deeper
				currentNode = c
				continue
			}
			// Create child node
			n := newNode(nodeT, insertPath, currentNode, nil, new(methodHandler), realPath, matchParam)
			n.addHandler(method, h)
			currentNode.addChild(n)
		} else {
			// 节点已经存在 只是添加不同的methodHandler
			if h != nil {
				currentNode.addHandler(method, h)
				currentNode.realPath = realPath
				if len(currentNode.matchParam) == 0 {
					currentNode.matchParam = matchParam
				}
			}
		}

		break
	}
}

func (r *Router) Find(method, path string, c Context) {
	var (
		childNode    *node
		paramCount   int
		nextNodeType nodeType
		nextNode     *node
		nextSearch   string
	)

	ctx := c.(*httpContext)
	ctx.path = path
	currentNode := r.tree
	search := path

	// 查找优先级 staticNodeType > paramNodeType > anyNodeType
	for {
		if search == "" {
			break
		}

		prefixLen := 0
		searchIndex := 0

		if currentNode.firstChar != ':' {
			searchLen := len(search)
			prefixLen = len(currentNode.prefix)
			maxLen := utils.MinInt(prefixLen, searchLen)

			for ; searchIndex < maxLen && search[searchIndex] == currentNode.prefix[searchIndex]; searchIndex++ {
			}
		}

		if searchIndex == prefixLen {
			search = search[searchIndex:]
		} else {
			currentNode = nextNode
			search = nextSearch
			if nextNodeType == paramNodeType {
				goto Param
			} else if nextNodeType == anyNodeType {
				goto Any
			}
			// Not found
			return
		}

		if search == "" {
			break
		}

		// staticNodeType
		if childNode = currentNode.findChild(search[0], staticNodeType); childNode != nil {
			if currentNode.prefix[len(currentNode.prefix)-1] == '/' {
				nextNodeType = paramNodeType
				nextNode = currentNode
				nextSearch = search
			}
			currentNode = childNode
			continue
		}

		// paramNodeType
	Param:
		if childNode = currentNode.findChildByNodeType(paramNodeType); childNode != nil {
			if len(ctx.paramValues) == paramCount {
				continue
			}

			if currentNode.prefix[len(currentNode.prefix)-1] == '/' {
				nextNodeType = anyNodeType
				nextNode = currentNode
				nextSearch = search
			}

			currentNode = childNode
			i, l := 0, len(search)
			for ; i < l && search[i] != '/'; i++ {
			}
			ctx.paramValues[paramCount] = search[:i]
			paramCount++
			search = search[i:]
			continue
		}

		// anyNodeType
	Any:
		if currentNode = currentNode.findChildByNodeType(anyNodeType); currentNode == nil {
			if nextNode != nil {
				currentNode = nextNode
				nextNode = currentNode.parent
				search = nextSearch
				if nextNodeType == paramNodeType {
					goto Param
				} else if nextNodeType == anyNodeType {
					goto Any
				}
			}
			// Not found
			return
		}
		ctx.paramValues[len(currentNode.matchParam)-1] = search
		break
	}

	ctx.handler = currentNode.findHandler(method)
	ctx.path = currentNode.realPath
	ctx.paramNames = currentNode.matchParam

	// 没有查到路由
	if ctx.handler == nil {
		ctx.handler = currentNode.checkMethodNotAllowed()

		// 再查下这个节点的子节点有没有* 可以匹配
		if currentNode = currentNode.findChildByNodeType(anyNodeType); currentNode != nil {
			if h := currentNode.findHandler(method); h != nil {
				ctx.handler = h
			} else {
				ctx.handler = currentNode.checkMethodNotAllowed()
			}
			ctx.path = currentNode.realPath
			ctx.paramNames = currentNode.matchParam
			ctx.paramValues[len(currentNode.matchParam)-1] = ""
		}
	}

	return
}

func newNode(nodeT nodeType, prefix string, parent *node, children []*node, mh *methodHandler,
	realPath string, matchParam []string) *node {
	return &node{
		nodeType:   nodeT,
		firstChar:  prefix[0],
		prefix:     prefix,
		parent:     parent,
		children:   children,
		realPath:   realPath,
		matchParam: matchParam,
		handlers:   mh,
	}
}

func (n *node) addHandler(method string, h HandlerFunc) {
	switch method {
	case CONNECT:
		n.handlers.connect = h
	case DELETE:
		n.handlers.delete = h
	case GET:
		n.handlers.get = h
	case HEAD:
		n.handlers.head = h
	case OPTIONS:
		n.handlers.options = h
	case PATCH:
		n.handlers.patch = h
	case POST:
		n.handlers.post = h
	case PUT:
		n.handlers.put = h
	case TRACE:
		n.handlers.trace = h
	}
}

func (n *node) findHandler(method string) HandlerFunc {
	switch method {
	case CONNECT:
		return n.handlers.connect
	case DELETE:
		return n.handlers.delete
	case GET:
		return n.handlers.get
	case HEAD:
		return n.handlers.head
	case OPTIONS:
		return n.handlers.options
	case PATCH:
		return n.handlers.patch
	case POST:
		return n.handlers.post
	case PUT:
		return n.handlers.put
	case TRACE:
		return n.handlers.trace
	default:
		return nil
	}
}

func (n *node) addChild(c *node) {
	n.children = append(n.children, c)
}

func (n *node) findChild(firstChar byte, nodeT nodeType) *node {
	for _, c := range n.children {
		if c.firstChar == firstChar && c.nodeType == nodeT {
			return c
		}
	}
	return nil
}

func (n *node) findChildByNodeType(nodeT nodeType) *node {
	for _, c := range n.children {
		if c.nodeType == nodeT {
			return c
		}
	}
	return nil
}

func (n *node) findChildWithFirstChar(firstChar byte) *node {
	for _, c := range n.children {
		if c.firstChar == firstChar {
			return c
		}
	}
	return nil
}

func (n *node) checkMethodNotAllowed() HandlerFunc {
	for _, m := range allowMethods {
		if h := n.findHandler(m); h != nil {
			return MethodNotAllowedHandler
		}
	}
	return NotFoundHandler
}
