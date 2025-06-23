package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator" // Ważne: do bardziej szczegółowego listowania portów
)

func main() {
	// --- Definicja flag (argumentów) ---
	// Nazwa portu: flaga "-port", domyślnie "" (pusta, co oznacza, że będziemy prosić o wybór)
	portNamePtr := flag.String("port", "", "Nazwa portu szeregowego (np. /dev/ttyUSB0, COM1). Jeśli puste, wyświetli listę.")
	// Prędkość transmisji (baud rate): flaga "-baud", domyślnie 9600
	baudRatePtr := flag.Int("baud", 9600, "Prędkość transmisji (baud rate), np. 9600, 115200")
	// Wiadomość do wysłania: flaga "-msg", domyślnie "Witaj PMOD!\n"
	messagePtr := flag.String("msg", "Witaj PMOD!\n", "Wiadomość do wysłania przez port szeregowy")

	// Parsowanie argumentów z wiersza poleceń
	flag.Parse()

	// Pobranie wartości z pointerów
	chosenPortName := *portNamePtr
	baudRate := *baudRatePtr
	message := *messagePtr

	// --- Wyświetlanie listy dostępnych portów i wybór przez użytkownika ---
	if chosenPortName == "" {
		fmt.Println("Skanowanie dostępnych portów szeregowych...")
		ports, err := enumerator.GetDetailedPortsList() // Używamy GetDetailedPortsList dla więcej informacji
		if err != nil {
			log.Fatalf("Błąd podczas listowania portów: %v", err)
		}

		if len(ports) == 0 {
			log.Fatalf("Nie znaleziono żadnych portów szeregowych! Upewnij się, że PMOD jest podłączony.")
		}

		fmt.Println("\nDostępne porty szeregowe:")
		for i, p := range ports {
			fmt.Printf("%d. %s", i+1, p.Name)
			if p.IsUSB {
				fmt.Printf(" (USB VID:%s PID:%s Serial:%s Product:%s)",
					p.VID, p.PID, p.SerialNumber, p.Product)
			}
			fmt.Println()
		}

		fmt.Printf("Wybierz numer portu (1-%d) lub wpisz nazwę portu ręcznie: ", len(ports))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Spróbuj przekonwertować na numer
		selectedIndex, err := strconv.Atoi(input)
		if err == nil && selectedIndex >= 1 && selectedIndex <= len(ports) {
			chosenPortName = ports[selectedIndex-1].Name
		} else {
			// Jeśli to nie numer, traktuj jako nazwę portu
			chosenPortName = input
			// Dodatkowa walidacja, czy podana nazwa istnieje na liście
			found := false
			for _, p := range ports {
				if p.Name == chosenPortName {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Ostrzeżenie: Podana nazwa portu '%s' nie znajduje się na liście wykrytych portów. Mimo to spróbuję otworzyć.\n", chosenPortName)
			}
		}

		if chosenPortName == "" {
			log.Fatal("Nie wybrano portu. Program zostanie zakończony.")
		}
	}

	fmt.Printf("Wybrany port: %s\n", chosenPortName)

	// --- Otwarcie portu szeregowego ---
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(chosenPortName, mode)
	if err != nil {
		log.Fatalf("Błąd podczas otwierania portu %s: %v\nUpewnij się, że nazwa portu jest poprawna i masz do niego uprawnienia.", chosenPortName, err)
	}
	defer port.Close() // Upewnij się, że port zostanie zamknięty po zakończeniu działania programu

	fmt.Printf("Pomyślnie otwarto port szeregowy %s z prędkością %d baud.\n", chosenPortName, baudRate)

	// --- Wysyłanie danych ---
	n, err := port.Write([]byte(message))
	if err != nil {
		log.Fatalf("Błąd podczas wysyłania danych: %v", err)
	}
	fmt.Printf("Wysłano %d bajtów: \"%s\"\n", n, message)

	// Krótka pauza, aby dać czas urządzeniu na odpowiedź
	time.Sleep(500 * time.Millisecond)

	// --- Odbieranie danych ---
	fmt.Println("Oczekiwanie na dane z PMOD (max 5 sekund)...")
	buff := make([]byte, 100) // Bufor na odczytane dane
	readAttempts := 0
	maxReadAttempts := 10 // Spróbuj odczytać 10 razy co 0.5 sekundy (łącznie 5 sekund)

	for readAttempts < maxReadAttempts {
		numRead, err := port.Read(buff)
		if err != nil {
			log.Fatalf("Błąd podczas odczytu danych: %v", err)
		}
		if numRead > 0 {
			receivedData := string(buff[:numRead])
			fmt.Printf("Odebrano %d bajtów: \"%s\"\n", numRead, receivedData)
			break // Jeśli coś odebraliśmy, wychodzimy
		}
		time.Sleep(500 * time.Millisecond)
		readAttempts++
	}

	if readAttempts == maxReadAttempts {
		fmt.Println("Nie odebrano danych w określonym czasie.")
	}

	fmt.Println("Program zakończony.")
}