package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type SRegistry struct{
	timeout time.Duration
	mu sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct{
	Addr string
	start time.Time
}

const(
	defaultPath = "/_srpc/registry"
	defaultTimeout=time.Minute *5
)

func New(timeout time.Duration) *SRegistry{
	return &SRegistry{
		servers: make(map[string]*ServerItem),
		timeout:timeout,
	}

}

var DefaultGeeRegister = New(defaultTimeout)

func (r *SRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s:=r.servers[addr]
	if s==nil{
		r.servers[addr] = &ServerItem{Addr: addr,start:time.Now()}
	}else{
		s.start = time.Now()
	}
}

func (r *SRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	alive := make([]string,0)

	for addr,s:=range r.servers{
		if r.timeout==0 || s.start.Add(r.timeout).After(time.Now()){
			alive=append(alive,addr)
		}else {
			delete(r.servers,addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *SRegistry) ServeHTTP(w http.ResponseWriter,req *http.Request)  {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Srpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr:=req.Header.Get("X-Srpc-Server")
		if addr==""{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *SRegistry) HandleHTTP(registryPath string){
	http.Handle(registryPath,r)
	log.Println("rpc registry path:", registryPath)
}
func HandleHTTP(){
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
func Heartbeat(registry,addr string,duration time.Duration) {
	if duration==0{
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err=sendHeartbeat(registry,addr)

	go func() {
		t:=time.NewTicker(duration)
		for err==nil{
			<-t.C
			err=sendHeartbeat(registry,addr)
		}
	}()

}
func sendHeartbeat(registry,addr string) error{
	log.Println(addr, "send heart beat to registry", registry)
	httpClient:=&http.Client{}
	req,_:=http.NewRequest("POST",registry,nil)
	req.Header.Set("X-Srpc-Server", addr)
	if _,err:=httpClient.Do(req);err!=nil{
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}

