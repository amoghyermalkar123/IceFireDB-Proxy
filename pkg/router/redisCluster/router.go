package redisCluster

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/IceFireDB/IceFireDB-Proxy/pkg/RedSHandle"
	"github.com/IceFireDB/IceFireDB-Proxy/pkg/rediscluster"
	"github.com/IceFireDB/IceFireDB-Proxy/pkg/router"
)

/**
 * 注册的命令列表
 */
func NewRouter(cluster *rediscluster.Cluster) *Router {
	r := &Router{
		redisCluster: cluster,
		cmd:          make(map[string]router.HandlersChain),
	}
	r.pool.New = func() interface{} {
		return r.allocateContext()
	}
	return r
}

const CMDEXEC = "CMDEXEC"

func (r *Router) InitCMD() {
	r.AddCommand("WCONFIG", r.cmdWCONFIG)
	r.AddCommand("COMMAND", r.cmdCOMMAND)
	r.AddCommand("PING", r.cmdPING)
	r.AddCommand("QUIT", r.cmdQUIT)
	r.AddCommand(CMDEXEC, r.cmdCMDEXEC)

	r.AddCommand("DEL", r.cmdDEL)
	r.AddCommand("EXISTS", r.cmdEXISTS)
	r.AddCommand("MGET", r.cmdMGET)
}

func (r *Router) Handle(w *RedSHandle.WriterHandle, args []interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Error("handle panic", r)
		}
	}()
	cmdType := strings.ToUpper(string(args[0].([]byte)))

	op, ok := router.OpTable[cmdType]

	if !ok || op.Flag.IsNotAllowed() {
		return router.WriteError(w, fmt.Errorf(router.ErrUnknownCommand, cmdType))
	}

	if !op.ArgsVerify(len(args)) {
		return router.WriteError(w, fmt.Errorf(router.ErrArguments, cmdType))
	}

	handlers, ok := r.cmd[cmdType]
	if !ok {
		handlers = r.cmd[CMDEXEC]
	}
	c := r.pool.Get().(*router.Context)
	defer func() {
		c.Reset()
		r.pool.Put(c)
	}()
	c.Index = -1
	c.Writer = w
	c.Args = args
	c.Handlers = handlers
	c.Cmd = cmdType
	c.Op = op.Flag
	c.Reply = nil

	return c.Next()
}

var _ router.IRoutes = (*Router)(nil)

type Router struct {
	redisCluster *rediscluster.Cluster
	MiddleWares  router.HandlersChain
	cmd          map[string]router.HandlersChain
	pool         sync.Pool
}

func (r *Router) Use(funcs ...router.HandlerFunc) router.IRoutes {
	r.MiddleWares = append(r.MiddleWares, funcs...)
	return r
}

func (r *Router) AddCommand(operation string, handlers ...router.HandlerFunc) router.IRoutes {
	handlers = r.combineHandlers(handlers)
	r.addRoute(operation, handlers)
	return r
}

func (r *Router) Close() error {
	r.redisCluster.Close()
	return nil
}

func (r *Router) addRoute(operation string, handlers router.HandlersChain) {
	if r.cmd == nil {
		r.cmd = make(map[string]router.HandlersChain)
	}
	r.cmd[operation] = handlers
}

func (r *Router) combineHandlers(handlers router.HandlersChain) router.HandlersChain {
	finalSize := len(r.MiddleWares) + len(handlers)
	if finalSize >= int(router.AbortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make(router.HandlersChain, finalSize)
	copy(mergedHandlers, r.MiddleWares)
	copy(mergedHandlers[len(r.MiddleWares):], handlers)
	return mergedHandlers
}

func (engine *Router) allocateContext() *router.Context {
	return &router.Context{}
}
