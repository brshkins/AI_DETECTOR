#include <iostream>
#include <string>
#include <iomanip>
#include <stdexcept>
using namespace std;

template <typename T>
class Vector {
private:
    T* elements;
    int capacity;
    int size;
    void resize();
public:
    Vector();
    ~Vector();
    Vector(const Vector&) = delete;
    Vector& operator=(const Vector&) = delete;
    void add(const T value);
    bool remove(int index);
    T& operator[](int index);
    bool update(int index, const T& value);
    bool lsh(int num);
    bool rsh(int num);
    void print();
    bool get(int index, T& value);
};

template <typename T>
void Vector<T>::resize() {
    if (size >= capacity) {
        int newCapacity;
        if (capacity == 0) {
            newCapacity = 10;
        }
        else {
            newCapacity = capacity * 2;
        }
        T* newElements = new T[newCapacity];
        for (int i = 0; i < size; i++) {
            newElements[i] = elements[i];
        }
        delete[] elements;
        elements = newElements;
        capacity = newCapacity;
    }
}

template <typename T>
Vector<T>::Vector() : elements(nullptr), capacity(0), size(0) {}

template <typename T>
Vector<T>::~Vector() {
    delete[] elements;
}

template <typename T>
void Vector<T>::add(const T value) {
    resize();
    elements[size] = value;
    size++;
}

template <typename T>
bool Vector<T>::remove(int index) {
    if (index < 0 || index >= size) {
        return false;
    }
    for (int i = index; i < size - 1; i++) {
        elements[i] = elements[i + 1];
    }
    size--;
    return true;
}

template <typename T>
T& Vector<T>::operator[](int index) {
    if (index < 0 || index >= size) {
        throw out_of_range("Index out of range");
    }
    return elements[index];
}

template <typename T>
bool Vector<T>::update(int index, const T& value) {
    if (index < 0 || index >= size) {
        return false;
    }
    elements[index] = value;
    return true;
}

template <typename T>
bool Vector<T>::lsh(int num) {
    if (size == 0) return true;
    if (num < 0) return false;
    num = num % size;
    if (num == 0) return true;
    auto reverse = [this](int start, int end) {
        while (start < end) {
            std::swap(elements[start], elements[end]);
            start++;
            end--;
        }
        };
    reverse(0, num - 1);
    reverse(num, size - 1);
    reverse(0, size - 1);
    return true;
}

template <typename T>
bool Vector<T>::rsh(int num) {
    if (size == 0) return true;
    if (num < 0) return false;
    num = num % size;
    if (num == 0) return true;
    return lsh(size - num);
}

template <typename T>
void Vector<T>::print() {
    for (int i = 0; i < size; i++) {
        cout << elements[i];
        if (i < size - 1) cout << "\n";
    }
    if (size > 0) cout << "\n";
}

template <typename T>
bool Vector<T>::get(int index, T& value) {
    if (index < 0 || index >= size) {
        return false;
    }
    value = elements[index];
    return true;
}

template <>
void Vector<double>::print() {
    for (int i = 0; i < size; i++) {
        cout << fixed << setprecision(2) << elements[i];
        if (i < size - 1) cout << "\n";
    }
    if (size > 0) cout << "\n";
}

int main() {
    char type;
    cin >> type;
    int N;
    cin >> N;

    if (type == 'I') {
        Vector<int> arr;
        for (int i = 0; i < N; i++) {
            string command;
            cin >> command;
            if (command == "ADD") {
                int value;
                cin >> value;
                arr.add(value);
            }
            else if (command == "REMOVE") {
                int index;
                cin >> index;
                if (!arr.remove(index)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "PRINT") {
                int index;
                cin >> index;
                int value;
                if (!arr.get(index, value)) {
                    cout << "ERROR\n";
                }
                else {
                    cout << value << "\n";
                }
            }
            else if (command == "UPDATE") {
                int index, value;
                cin >> index >> value;
                if (!arr.update(index, value)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "LSH") {
                int num;
                cin >> num;
                if (!arr.lsh(num)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "RSH") {
                int num;
                cin >> num;
                if (!arr.rsh(num)) {
                    cout << "ERROR\n";
                }
            }
        }
        arr.print();
    }
    else if (type == 'D') {
        Vector<double> arr;
        for (int i = 0; i < N; i++) {
            string command;
            cin >> command;
            if (command == "ADD") {
                double value;
                cin >> value;
                arr.add(value);
            }
            else if (command == "REMOVE") {
                int index;
                cin >> index;
                if (!arr.remove(index)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "PRINT") {
                int index;
                cin >> index;
                double value;
                if (!arr.get(index, value)) {
                    cout << "ERROR\n";
                }
                else {
                    cout << fixed << setprecision(2) << value << "\n";
                }
            }
            else if (command == "UPDATE") {
                int index;
                double value;
                cin >> index >> value;
                if (!arr.update(index, value)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "LSH") {
                int num;
                cin >> num;
                if (!arr.lsh(num)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "RSH") {
                int num;
                cin >> num;
                if (!arr.rsh(num)) {
                    cout << "ERROR\n";
                }
            }
        }
        arr.print();
    }
    else if (type == 'S') {
        Vector<string> arr;
        for (int i = 0; i < N; i++) {
            string command;
            cin >> command;
            if (command == "PRINT") {
                int index;
                cin >> index;
                string value;
                if (!arr.get(index, value)) {
                    cout << "ERROR\n";
                }
                else {
                    cout << value << "\n";
                }
            }
            else if (command == "ADD") {
                string value;
                cin >> value;
                arr.add(value);
            }
            else if (command == "REMOVE") {
                int index;
                cin >> index;
                if (!arr.remove(index)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "UPDATE") {
                int index;
                string value;
                cin >> index >> value;
                if (!arr.update(index, value)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "LSH") {
                int num;
                cin >> num;
                if (!arr.lsh(num)) {
                    cout << "ERROR\n";
                }
            }
            else if (command == "RSH") {
                int num;
                cin >> num;
                if (!arr.rsh(num)) {
                    cout << "ERROR\n";
                }
            }
        }
        arr.print();
    }
    return 0;
}

