package rabbitmq

import (
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

type Channel struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewChannel() *Channel {
	return &Channel{}
}

func (c *Channel) Connect() (Channel, error) {
	username := os.Getenv("RABBITMQ_USERNAME")
	password := os.Getenv("RABBITMQ_PASSWORD")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")
	vhost := os.Getenv("RABBITMQ_VHOST")

	var err error
	// for {
	// 	c.Conn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, password, host, port, vhost))
	// 	if err == nil {
	// 		break
	// 	}
	// 	log.Println("Failed to connect to RabbitMQ, retrying in 5 seconds...")
	// 	time.Sleep(5 * time.Second)
	// }

	c.Conn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, password, host, port, vhost))
	if err != nil {
		log.Println("Failed to connect to RabbitMQ")
		return *c, err
	}

	c.Channel, err = c.Conn.Channel()
	if err != nil {
		return *c, err
	}

	return *c, nil
}

func (c *Channel) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.Conn != nil {
		c.Conn.Close()
	}
}

// func NewChannel() *amqp.Channel {
// 	username := os.Getenv("RABBITMQ_USERNAME")
// 	password := os.Getenv("RABBITMQ_PASSWORD")
// 	host := os.Getenv("RABBITMQ_HOST")
// 	port := os.Getenv("RABBITMQ_PORT")
// 	vhost := os.Getenv("RABBITMQ_VHOST")

// 	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, password, host, port, vhost))
// 	if err != nil {
// 		log.Printf("failed to connect to RabbitMQ: %v", err)
// 		return nil
// 	}

// 	ch, err := conn.Channel()
// 	if err != nil {
// 		log.Printf("failed to open a RabbitMQ channel: %v", err)
// 		return nil
// 	}

// 	return ch
// }
