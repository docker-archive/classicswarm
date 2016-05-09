package keystone

//	"github.com/garyburd/redigo/redis"

type Cache struct{}

//var pool = newPool()

func (*Cache) Get(key string) string {
	//	c := pool.Get()
	//	defer c.Close()
	//	response, _ := redis.String(c.Do("GET", key))
	//	log.Println("Redis returns: ")
	//	log.Println(response)
	//	return response
	return ""
}

func (*Cache) PutEx(key, val string, ex int64) {
	//	c := pool.Get()
	//	defer c.Close()

	//	//Minimum 1 hour
	//	if ex < 3600 {
	//		ex = 3600
	//	}

	//	response, _ := c.Do("SETEX", key, ex, val)
	//	log.Println("Put Redis responded")
	//	log.Println(response)
}

func newPool() { //*redis.Pool {
	//	return &redis.Pool{
	//		MaxIdle:   80,
	//		MaxActive: 12000, // max number of connections
	//		Dial: func() (redis.Conn, error) {
	//			c, err := redis.Dial("tcp", ":6379")
	//			if err != nil {
	//				log.Println("Error while initializing Redis connection...")
	//				//				log.Fatal(err)
	//			} else {
	//				log.Println("Redis OK ")
	//			}
	//			return c, err
	//		},
	//	}
}

func (*Cache) Init() {
	//	c := pool.Get()
	//	defer c.Close()
	//	test, err := c.Do("HGETALL", "test:1")
	//	if err != nil {
	//		log.Println("Seems redis is not reachable")
	//	}

	//	log.Println(test)
}
