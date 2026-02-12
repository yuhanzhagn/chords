function TestBackendButton() {
  const handleClick = async () => {
    const storedToken = localStorage.getItem('jwt');

    if (!storedToken) {
      alert('No JWT token found in localStorage!');
      return;
    }

    try {
      const response = await fetch('http://13.158.200.71:8080/chatrooms/1/messages', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${storedToken}`,
        },
      });

      const data = await response.json();
      console.log('Backend response:', data);
      alert('Backend responded, check console for details.');
    } catch (error: unknown) {
      console.error('Error connecting to backend:', error);
      alert('Failed to connect to backend. Check console.');
    }
  };

  return <button onClick={handleClick}>Test Backend Connection</button>;
}

export default TestBackendButton;
